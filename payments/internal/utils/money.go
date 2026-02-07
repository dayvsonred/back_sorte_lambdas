package utils

import (
	"errors"
	"strconv"
	"strings"
)

func ParseAmountToCents(amount string) (int64, error) {
	value := strings.TrimSpace(amount)
	if value == "" {
		return 0, errors.New("amount vazio")
	}

	if strings.Contains(value, ",") && strings.Contains(value, ".") {
		return 0, errors.New("use apenas um separador decimal")
	}

	if strings.Contains(value, ",") {
		value = strings.ReplaceAll(value, ",", ".")
	}

	if strings.Contains(value, ".") {
		parts := strings.Split(value, ".")
		if len(parts) != 2 {
			return 0, errors.New("formato decimal invalido")
		}
		intPart := parts[0]
		fracPart := parts[1]
		if intPart == "" {
			intPart = "0"
		}
		if !isDigits(intPart) || !isDigits(fracPart) {
			return 0, errors.New("valor deve conter apenas digitos")
		}
		if len(fracPart) > 2 {
			return 0, errors.New("use no maximo 2 casas decimais")
		}
		if len(fracPart) == 1 {
			fracPart = fracPart + "0"
		}
		if len(fracPart) == 0 {
			fracPart = "00"
		}

		intVal, err := strconv.ParseInt(intPart, 10, 64)
		if err != nil {
			return 0, errors.New("parte inteira invalida")
		}
		fracVal, err := strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return 0, errors.New("parte decimal invalida")
		}

		cents := intVal*100 + fracVal
		if cents <= 0 {
			return 0, errors.New("amount deve ser maior que zero")
		}
		return cents, nil
	}

	if !isDigits(value) {
		return 0, errors.New("valor deve conter apenas digitos")
	}
	cents, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errors.New("valor invalido")
	}
	if cents <= 0 {
		return 0, errors.New("amount deve ser maior que zero")
	}
	return cents, nil
}

func isDigits(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
