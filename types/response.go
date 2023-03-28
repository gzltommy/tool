package types

type (
	PageResult struct {
		Page  int         `json:"page"`
		Limit int         `json:"limit"`
		Items interface{} `json:"items"`
		Total int64       `json:"total"`
	}
)
