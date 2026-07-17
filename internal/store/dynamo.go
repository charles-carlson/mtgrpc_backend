package store

import (
	"context"
	"fmt"

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
	Rarity []string
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
	Rarity   string   `json:"rarity"    dynamodbav:"rarity"`
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

	colors, err := attributevalue.Marshal(card.Colors)
	if err != nil {
		return err
	}

	rarity, err := attributevalue.Marshal(card.Rarity)
	if err != nil {
		return err
	}

	_, err = s.db.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(TableName),
		Key:       key,
		UpdateExpression: aws.String(
			"ADD #count :delta SET #set = if_not_exists(#set, :set), #number = if_not_exists(#number, :number), image_url = if_not_exists(image_url, :image_url), prices = :prices, colors = :colors, rarity = :rarity",
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
			":colors":    colors,
			":rarity":    rarity,
		},
	})
	return err
}

// ScanAllCards returns the entire collection, following pagination to the end.
// It feeds the in-memory snapshot (see internal/cards) and price refresh.
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
