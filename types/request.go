package types

type (
	ReqPage struct {
		Page  int `form:"page" json:"page" binding:"required,gte=1"`
		Limit int `form:"limit" json:"limit" binding:"required,gte=1,lte=100"`
	}
)
