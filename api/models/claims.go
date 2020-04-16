package models

type Claims struct {
	Sub				string		`json:"sub"`
	ITLabInterface	interface{} `json:"itlab"`
	ITLab			[]string
}
