package donation

import (
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var jwtSecretKey1 = []byte("SUA_CHAVE_SECRETA")

func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, transform.RemoveFunc(isMn), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func isMn(r rune) bool {
	return unicode.Is(unicode.Mn, r)
}

func generateUniqueLinkName(storeDDB *dynamo.Store, base string) (string, error) {
	base = strings.ToLower(base)
	base = removeAccents(base)
	base = strings.ReplaceAll(base, " ", "_")

	re := regexp.MustCompile(`[^a-z0-9_]+`)
	base = re.ReplaceAllString(base, "")

	link := "@" + base
	finalLink := link

	letters := "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(time.Now().UnixNano())

	for {
		out, err := storeDDB.Query(context.Background(), &dynamodb.QueryInput{
			KeyConditionExpression: aws.String("PK = :pk"),
			ExpressionAttributeValues: map[string]types.AttributeValue{
				":pk": dynamo.S(store.LinkPK(finalLink)),
			},
			Limit: aws.Int32(1),
		})
		if err != nil {
			return "", err
		}
		if len(out.Items) == 0 {
			break
		}
		finalLink = fmt.Sprintf("%s_%c", link, letters[rand.Intn(len(letters))])
	}

	return finalLink, nil
}
