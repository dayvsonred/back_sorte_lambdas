package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv carrega as variÃ¡veis de ambiente do arquivo .env
func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Arquivo .env nÃ£o encontrado, usando variÃ¡veis de ambiente padrÃ£o.")
	}
}

// GetDatabaseURL retorna a URL de conexÃ£o com o banco de dados
func GetDatabaseURL() string {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL nÃ£o definida nas variÃ¡veis de ambiente.")
	}
	return dbURL
}

// GetDatabaseURL retorna a URL de conexÃ£o com o banco de dados
func GetPortServerStart() string {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		return "8080"
	}
	return port
}

func Getclient_id() string {
	client_id := os.Getenv("CLIENT_ID")
	if client_id == "" {
		log.Fatal("CLIENT_ID nÃ£o definida nas variÃ¡veis de ambiente.")
	}
	return client_id
}

func GetCredentials() map[string]interface{} {
	// Converte SANDBOX para booleano
	sandbox, err := strconv.ParseBool(os.Getenv("SANDBOX"))
	if err != nil {
		log.Printf("Erro ao converter SANDBOX para booleano: %v. Usando false como padrÃ£o.", err)
		sandbox = false
	}

	// Converte TIMEOUT para inteiro
	timeout, err := strconv.Atoi(os.Getenv("TIMEOUT"))
	if err != nil {
		log.Printf("Erro ao converter TIMEOUT para inteiro: %v. Usando 30 como padrÃ£o.", err)
		timeout = 30 // Valor padrÃ£o para TIMEOUT
	}

	return map[string]interface{}{
		"client_id":     os.Getenv("CLIENT_ID"),
		"client_secret": os.Getenv("CLIENT_SECRET"),
		"sandbox":       sandbox, // Agora Ã© um booleano
		"timeout":       timeout,
		"CA":            os.Getenv("CA_PEM"),
		"Key":           os.Getenv("KEY_PEM"),
	}
}

func GetAwsRegion() string {
	return os.Getenv("AWS_REGION")
}

func GetAwsAccessKey() string {
	return os.Getenv("AWS_ACCESS_KEY_ID")
}

func GetAwsSecretKey() string {
	return os.Getenv("AWS_SECRET_ACCESS_KEY")
}

func GetAwsBucket() string {
	return os.Getenv("AWS_BUCKET_NAME")
}

func GetJwtSecret() string {
	return os.Getenv("JWT_SECRET")
}

func GetPasswordResetKey() string {
	return os.Getenv("PASSWORD_RESET_KEY")
}

func GetawsBucketNameImgDoacao() string {
	return os.Getenv("AWS_BUCKET_NAME_IMG_DOACAO")
}

func GetDynamoTableName() string {
	return os.Getenv("DYNAMODB_TABLE")
}

