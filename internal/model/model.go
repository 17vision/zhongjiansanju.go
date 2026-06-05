package model

type PaginateReq struct {
	Page     int `json:"page" v:"required#页码不能为空"`
	PageSize int `json:"pageSize" v:"required#每页条数不能为空"`
}

type PaginateRes struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type Result struct {
	Result  bool   `json:"result"`
	Message string `json:"message"`
}
