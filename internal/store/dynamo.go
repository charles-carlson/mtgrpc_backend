package store

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// SearchFilter holds optional fields for SearchCards.
// Only non-empty fields are applied as filter conditions.
type SearchFilter struct {
	Name   string
	Set    string
	Colors []string
}

const TableName = "cards"

// Prices holds Scryfall market prices. All values are strings (e.g. "0.15") or empty if unavailable.
type Prices struct {
	USD     string `json:"usd"      dynamodbav:"usd"`
	USDFoil string `json:"usd_foil" dynamodbav:"usd_foil"`
	EUR     string `json:"eur"      dynamodbav:"eur"`
	EURFoil string `json:"eur_foil" dynamodbav:"eur_foil"`
	TIX     string `json:"tix"      dynamodbav:"tix"`
}

// Card mirrors the Manabox export shape.
// PK = Name, SK = Set#Number
type Card struct {
	Name     string   `json:"name"      dynamodbav:"name"`
	Set      string   `json:"set"       dynamodbav:"set"`
	Number   string   `json:"number"    dynamodbav:"number"`
	Count    int      `json:"count"     dynamodbav:"count"`
	ImageURL string   `json:"image_url" dynamodbav:"image_url"`
	Prices   Prices   `json:"prices"    dynamodbav:"prices"`
	Colors   []string `json:"colors"    dynamodbav:"colors"`
}

func (c Card) sk() string {
	return fmt.Sprintf("%s#%s", c.Set, c.Number)
}

type Store struct {
	db *dynamodb.Client
}

func New(db *dynamodb.Client) *Store {
	return &Store{db: db}
}
func (s *Store) RemoveCard(ctx context.Context, card Card) error {
	key, err := attributevalue.MarshalMap(map[string]any{
		"name":       card.Name,
		"set_number": card.sk(),
	})
	if err != nil {
		return err
	}
	delta, err := attributevalue.Marshal(-card.Count)
	if err != nil {
		return err
	}
	out, err := s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:    aws.String(TableName),
		Key:          key,
		ReturnValues: types.ReturnValueUpdatedNew,
		UpdateExpression: aws.String(
			"ADD #count :delta",
		),
		ExpressionAttributeNames: map[string]string{
			"#count": "count",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":delta": delta,
		},
	})
	if err != nil {
		return err
	}
	var output int
	if err := attributevalue.Unmarshal(out.Attributes["count"], &output); err != nil {
		return err
	}
	if output <= 0 {
		_, err := s.db.DeleteItem(ctx, &dynamodb.DeleteItemInput{
			TableName: aws.String(TableName),
			Key:       key,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// PutCard writes a card to DynamoDB. If the card already exists, the count
// is atomically incremented rather than overwritten.
func (s *Store) PutCard(ctx context.Context, card Card) error {
	key, err := attributevalue.MarshalMap(map[string]any{
		"name":       card.Name,
		"set_number": card.sk(),
	})
	if err != nil {
		return err
	}

	delta, err := attributevalue.Marshal(card.Count)
	if err != nil {
		return err
	}

	imageURL, err := attributevalue.Marshal(card.ImageURL)
	if err != nil {
		return err
	}

	set, err := attributevalue.Marshal(card.Set)
	if err != nil {
		return err
	}

	number, err := attributevalue.Marshal(card.Number)
	if err != nil {
		return err
	}

	prices, err := attributevalue.Marshal(card.Prices)
	if err != nil {
		return err
	}

	_, err = s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(TableName),
		Key:       key,
		UpdateExpression: aws.String(
			"ADD #count :delta SET #set = if_not_exists(#set, :set), #number = if_not_exists(#number, :number), image_url = if_not_exists(image_url, :image_url), prices = :prices",
		),
		ExpressionAttributeNames: map[string]string{
			"#count":  "count",
			"#set":    "set",
			"#number": "number",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":delta":     delta,
			":set":       set,
			":number":    number,
			":image_url": imageURL,
			":prices":    prices,
		},
	})
	return err
}

// GetCard fetches a specific printing by name + set + number.
func (s *Store) GetCard(ctx context.Context, name, set, number string) (*Card, error) {
	key, err := attributevalue.MarshalMap(map[string]any{
		"name":       name,
		"set_number": fmt.Sprintf("%s#%s", set, number),
	})
	if err != nil {
		return nil, err
	}

	out, err := s.db.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key:       key,
	})
	if err != nil {
		return nil, err
	}
	if out.Item == nil {
		return nil, nil
	}

	var card Card
	if err := attributevalue.UnmarshalMap(out.Item, &card); err != nil {
		return nil, err
	}
	return &card, nil
}

// QueryByName returns all printings of a card across sets.
func (s *Store) QueryByName(ctx context.Context, name string) ([]Card, error) {
	nameKey, err := attributevalue.Marshal(name)
	if err != nil {
		return nil, err
	}

	out, err := s.db.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String(TableName),
		KeyConditionExpression: aws.String("#n = :name"),
		ExpressionAttributeNames: map[string]string{
			"#n": "name",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":name": nameKey,
		},
	})
	if err != nil {
		return nil, err
	}

	var cards []Card
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &cards); err != nil {
		return nil, err
	}
	return cards, nil
}

// ScanAll returns every card in the collection.
func (s *Store) ScanAll(ctx context.Context, pageSize int32, pageToken string) ([]Card, string, error) {
	startKey, err := decodePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
	input := &dynamodb.ScanInput{
		TableName: aws.String(TableName),
	}
	if pageSize > 0 {
		input.Limit = aws.Int32(pageSize)
	}
	if startKey != nil {
		input.ExclusiveStartKey = startKey
	}
	out, err := s.db.Scan(ctx, input)
	if err != nil {
		return nil, "", err
	}
	nextToken, err := encodePageToken(out.LastEvaluatedKey)
	if err != nil {
		return nil, "", err
	}
	var cards []Card
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &cards); err != nil {
		return nil, "", err
	}
	return cards, nextToken, nil
}

// Search scans the table applying only the non-empty fields in the filter.
// If all fields are empty it falls back to a full scan.
func (s *Store) Search(ctx context.Context, f SearchFilter, pageSize int32, pageToken string) ([]Card, string, error) {
	var (
		conditions []string
		attrNames  = map[string]string{}
		attrValues = map[string]types.AttributeValue{}
	)
	startKey, err := decodePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}
	if f.Name != "" {
		v, err := attributevalue.Marshal(f.Name)
		if err != nil {
			return nil, "", err
		}
		conditions = append(conditions, "#n = :name")
		attrNames["#n"] = "name"
		attrValues[":name"] = v
	}

	if f.Set != "" {
		v, err := attributevalue.Marshal(f.Set)
		if err != nil {
			return nil, "", err
		}
		conditions = append(conditions, "#s = :set")
		attrNames["#s"] = "set"
		attrValues[":set"] = v
	}

	for i, color := range f.Colors {
		v, err := attributevalue.Marshal(color)
		if err != nil {
			return nil, "", err
		}
		placeholder := fmt.Sprintf(":color%d", i)
		conditions = append(conditions, fmt.Sprintf("contains(colors, %s)", placeholder))
		attrValues[placeholder] = v
	}

	input := &dynamodb.QueryInput{
		TableName: aws.String(TableName),
		IndexName: aws.String("set-index"),
	}

	if len(conditions) > 0 {
		input.FilterExpression = aws.String(strings.Join(conditions, " AND "))
		input.ExpressionAttributeValues = attrValues
		if len(attrNames) > 0 {
			input.ExpressionAttributeNames = attrNames
		}
	}
	if pageSize > 0 {
		input.Limit = aws.Int32(pageSize)
	}
	if startKey != nil {
		input.ExclusiveStartKey = startKey
	}
	out, err := s.db.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	nextToken, err := encodePageToken(out.LastEvaluatedKey)
	if err != nil {
		return nil, "", err
	}
	var cards []Card
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &cards); err != nil {
		return nil, "", err
	}
	return cards, nextToken, nil
}

// QueryBySet returns all cards in a given set using a Scan with filter.
// set is not a key attribute so a Query is not possible without a GSI.
func (s *Store) QueryBySet(ctx context.Context, set string, pageSize int32, pageToken string) ([]Card, string, error) {
	startKey, err := decodePageToken(pageToken)
	if err != nil {
		return nil, "", err
	}

	setKey, err := attributevalue.Marshal(set)
	if err != nil {
		return nil, "", err
	}
	input := &dynamodb.QueryInput{
		TableName:        aws.String(TableName),
		IndexName:        aws.String("set-index"),
		FilterExpression: aws.String("#s = :set"),
		ExpressionAttributeNames: map[string]string{
			"#s": "set",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":set": setKey,
		},
	}
	if pageSize > 0 {
		input.Limit = aws.Int32(pageSize)
	}
	if startKey != nil {
		input.ExclusiveStartKey = startKey
	}
	out, err := s.db.Query(ctx, input)
	if err != nil {
		return nil, "", err
	}
	nextToken, err := encodePageToken(out.LastEvaluatedKey)
	if err != nil {
		return nil, "", err
	}
	var cards []Card
	if err := attributevalue.UnmarshalListOfMaps(out.Items, &cards); err != nil {
		return nil, "", err
	}
	return cards, nextToken, nil
}
func (s *Store) ScanAllCards(ctx context.Context) ([]Card, error) {
	var all []Card
	var startKey map[string]types.AttributeValue
	for {
		out, err := s.db.Scan(ctx, &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: startKey,
		})
		if err != nil {
			return nil, err
		}
		var page []Card
		if err := attributevalue.UnmarshalListOfMaps(out.Items, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		if out.LastEvaluatedKey == nil {
			break
		}
		startKey = out.LastEvaluatedKey
	}
	return all, nil
}
func (s *Store) UpdatePrices(ctx context.Context, card Card) error {
	key, err := attributevalue.MarshalMap(map[string]any{
		"name":       card.Name,
		"set_number": card.sk(),
	})
	if err != nil {
		return err
	}
	prices, err := attributevalue.Marshal(card.Prices)
	if err != nil {
		return err
	}
	_, err = s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:        aws.String(TableName),
		Key:              key,
		UpdateExpression: aws.String("SET prices = :prices"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":prices": prices,
		},
	})
	return err
}
func encodePageToken(key map[string]types.AttributeValue) (string, error) {
	if key == nil {
		return "", nil
	}
	b, err := json.Marshal(key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}
func decodePageToken(token string) (map[string]types.AttributeValue, error) {
	if token == "" {
		return nil, nil
	}
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	var key map[string]types.AttributeValue
	if err := json.Unmarshal(b, &key); err != nil {
		return nil, err
	}
	return key, nil
}
