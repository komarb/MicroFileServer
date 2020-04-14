package models

type Claims struct {
	Sub		string		`json:"sub"`
	ITLab	[]string	`json:"itlab"`
	Scope   []string	`json:"scope"`
}
