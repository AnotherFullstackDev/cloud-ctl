package services

type ListServicesInput struct {
}

type ListServicesResponse struct {
	Service []Service `json:"service"`
	Cursor  string    `json:"cursor"`
}
