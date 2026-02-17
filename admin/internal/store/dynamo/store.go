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

func (s *Store) UpdateItem(ctx context.Context, key map[string]types.AttributeValue, update string, names map[string]string, values map[string]types.AttributeValue) error {
	_, err := s.Client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName:                 &s.Table,
		Key:                       key,
		UpdateExpression:          &update,
		ExpressionAttributeNames:  names,
		ExpressionAttributeValues: values,
	})
	return err
}

func (s *Store) Query(ctx context.Context, input *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	input.TableName = &s.Table
	return s.Client.Query(ctx, input)
}

func (s *Store) Scan(ctx context.Context, input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	input.TableName = &s.Table
	return s.Client.Scan(ctx, input)
}

func (s *Store) TransactWrite(ctx context.Context, items []types.TransactWriteItem) error {
	_, err := s.Client.TransactWriteItems(ctx, &dynamodb.TransactWriteItemsInput{
		TransactItems: items,
	})
	return err
}

func (s *Store) BatchGet(ctx context.Context, keys []map[string]types.AttributeValue) (map[string]map[string]types.AttributeValue, error) {
	out, err := s.Client.BatchGetItem(ctx, &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			s.Table: {
				Keys: keys,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	items := map[string]map[string]types.AttributeValue{}
	for _, item := range out.Responses[s.Table] {
		pk := item["PK"].(*types.AttributeValueMemberS).Value
		sk := item["SK"].(*types.AttributeValueMemberS).Value
		items[pk+"|"+sk] = item
	}
	return items, nil
}
