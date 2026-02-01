package users

import (
	"BACK_SORTE_GO/config"
	"BACK_SORTE_GO/internal/store"
	"BACK_SORTE_GO/internal/store/dynamo"
	"BACK_SORTE_GO/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

func UploadUserProfileImageHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Token ausente", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtSecretKey, nil
		})
		if err != nil || !token.Valid {
			http.Error(w, "Token invalido", http.StatusUnauthorized)
			return
		}

		idFromToken, ok := claims["sub"].(string)
		if !ok || idFromToken == "" {
			http.Error(w, "ID do usuario invalido", http.StatusUnauthorized)
			return
		}

		err = r.ParseMultipartForm(10 << 20)
		if err != nil {
			http.Error(w, "Erro ao parsear o formulario: "+err.Error(), http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("image")
		if err != nil {
			http.Error(w, "Erro ao ler o arquivo: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" {
			http.Error(w, "Formato de imagem nao suportado", http.StatusBadRequest)
			return
		}

		fileName := idFromToken + ext
		bucket := config.GetAwsBucket()
		if bucket == "" {
			http.Error(w, "Configuracao do bucket nao encontrada", http.StatusInternalServerError)
			return
		}

		url, err := utils.UploadToS3(file, fileName, bucket)
		if err != nil {
			http.Error(w, "Erro ao fazer upload no S3: "+err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := r.Context()
		item := map[string]types.AttributeValue{
			"PK":          dynamo.S(store.UserPK(idFromToken)),
			"SK":          dynamo.S("DETAILS"),
			"id":          dynamo.S(uuid.NewString()),
			"id_user":     dynamo.S(idFromToken),
			"img_perfil":  dynamo.S(fileName),
			"date_update": dynamo.S(time.Now().Format(time.RFC3339)),
		}

		_ = storeDDB.PutItem(ctx, item)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		resp := map[string]string{"url": url}
		json.NewEncoder(w).Encode(resp)
	}
}

func UserProfileImageHandler(storeDDB *dynamo.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		userID := vars["id"]

		if userID == "" {
			http.Error(w, "ID do usuario e obrigatorio", http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		item, err := storeDDB.GetItem(ctx, store.UserPK(userID), "DETAILS")
		if err != nil || len(item) == 0 {
			http.Error(w, "Usuario nao encontrado ou sem imagem", http.StatusNotFound)
			return
		}
		imgAttr, ok := item["img_perfil"].(*types.AttributeValueMemberS)
		if !ok || imgAttr.Value == "" {
			http.Error(w, "Imagem de perfil nao cadastrada", http.StatusNotFound)
			return
		}

		region := config.GetAwsRegion()
		bucket := config.GetAwsBucket()
		if region == "" || bucket == "" {
			http.Error(w, "Configuracao do bucket nao encontrada", http.StatusInternalServerError)
			return
		}

		url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", bucket, region, imgAttr.Value)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"image_url": "%s"}`, url)))
	}
}
