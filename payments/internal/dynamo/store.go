package dynamo

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Store struct {
	Client *dynamodb.Client
	Table  string
}

func New(client *dynamodb.Client, table string) *Store {
	return &Store{Client: client, Table: table}
}

func (s *Store) TableName() string {
	return s.Table
}

func (s *Store) GetItem(ctx context.Context, pk, sk string) (map[string]types.AttributeValue, error) {
	out, err := s.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &s.Table,
		Key: map[string]types.AttributeValue{
			"PK": &types.AttributeValueMemberS{Value: pk},
			"SK": &types.AttributeValueMemberS{Value: sk},
		},
	})
	if err != nil {
		return nil, err
	}
	return out.Item, nil
}

func (s *Store) PutItem(ctx context.Context, item map[string]types.AttributeValue) error {
	_, err := s.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: &s.Table,
		Item:      item,
	})
	return err
}

func (s *Store) TransactWrite(ctx context.Context, items []types.TransactWriteItem) error {
	_, err := s.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: items,
	})
	return err
}
