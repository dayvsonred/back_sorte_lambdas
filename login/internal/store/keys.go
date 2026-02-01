package store

import "strings"

const (
	PrefixUser          = "USER#"
	PrefixDonation      = "DONATION#"
	PrefixLink          = "LINK#"
	PrefixPassword      = "PWDREC#"
	PrefixTx            = "TX#"
	PrefixContact       = "CONTACT#"
	PrefixBank          = "BANK#"
	PrefixVisualization = "VIS#"
	PrefixPix           = "PIX#"
)

func UserPK(id string) string {
	return PrefixUser + id
}

func DonationPK(id string) string {
	return PrefixDonation + id
}

func LinkPK(link string) string {
	return PrefixLink + link
}

func PasswordPK(email string) string {
	return PrefixPassword + strings.ToLower(email)
}

func TxPK(txid string) string {
	return PrefixTx + txid
}

func ContactPK(id string) string {
	return PrefixContact + id
}

func BankPK(id string) string {
	return PrefixBank + id
}
