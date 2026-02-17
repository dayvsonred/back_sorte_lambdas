package dynamo

import "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

func S(value string) types.AttributeValue {
	return &types.AttributeValueMemberS{Value: value}
}

func N(value string) types.AttributeValue {
	return &types.AttributeValueMemberN{Value: value}
}

func B(value bool) types.AttributeValue {
	return &types.AttributeValueMemberBOOL{Value: value}
}
