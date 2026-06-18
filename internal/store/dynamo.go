package store

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const TableName = "cards"

// Card mirrors the Manabox export shape.
// PK = Name, SK = Set#Number
type Card struct {
	Name     string `json:"name"      dynamodbav:"name"`
	Set      string `json:"set"       dynamodbav:"set"`
	Number   string `json:"number"    dynamodbav:"number"`
	Count    int    `json:"count"     dynamodbav:"count"`
	ImageURL string `json:"image_url" dynamodbav:"image_url"`
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

// PutCard writes a card to DynamoDB.
func (s *Store) PutCard(ctx context.Context, card Card) error {
	item, err := attributevalue.MarshalMap(map[string]any{
		"name":       card.Name,
		"set_number": card.sk(),
		"set":        card.Set,
		"number":     card.Number,
		"count":      card.Count,
		"image_url":  card.ImageURL,
	})
	if err != nil {
		return err
	}

	_, err = s.db.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
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
func (s *Store) ScanAll(ctx context.Context) ([]Card, error) {
	out, err := s.db.Scan(ctx, &dynamodb.ScanInput{
		TableName: aws.String(TableName),
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

// QueryBySet returns all cards in a given set using a Scan with filter.
// set is not a key attribute so a Query is not possible without a GSI.
func (s *Store) QueryBySet(ctx context.Context, set string) ([]Card, error) {
	setKey, err := attributevalue.Marshal(set)
	if err != nil {
		return nil, err
	}

	out, err := s.db.Scan(ctx, &dynamodb.ScanInput{
		TableName:        aws.String(TableName),
		FilterExpression: aws.String("#s = :set"),
		ExpressionAttributeNames: map[string]string{
			"#s": "set",
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":set": setKey,
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
