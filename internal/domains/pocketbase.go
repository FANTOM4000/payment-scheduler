package domains

import "github.com/shopspring/decimal"

type AuthResponse struct {
	Token  string          `json:"token"`
	Record SuperUserRecord `json:"record"`
}

type RequestConnectResponse struct {
	ClientId string `json:"clientId"`
}

type RecordHook[T any] struct {
	Action string `json:"action"`
	Record T      `json:"record"`
}

type ListRecordsResponse[T any] struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	TotalPages int `json:"totalPages"`
	TotalItems int `json:"totalItems"`
	Items      []T `json:"items"`
}

type PaymentRecord struct {
	Id          string          `json:"id"`
	UserId      string          `json:"userId"`
	PaymentType string          `json:"paymentType"`
	Amount      decimal.Decimal `json:"amount"`
	PhoneNumber string          `json:"phoneNumber"`
	Status      string          `json:"status"`
	PaymentUrl  string          `json:"paymentUrl"`
}

type CreateRecordResponse struct {
	CollectionId   string `json:"collectionId"`
	CollectionName string `json:"collectionName"`
	Id             string `json:"id"`
	Created        string `json:"created"`
	Updated        string `json:"updated"`
}

type SuperUserRecord struct {
	CollectionId    string `json:"collectionId"`
	CollectionName  string `json:"collectionName"`
	Id              string `json:"id"`
	Email           string `json:"email"`
	EmailVisibility bool   `json:"emailVisibility"`
	Verified        bool   `json:"verified"`
	Created         string `json:"created"`
	Updated         string `json:"updated"`
}

type RequestTimeForFreePostRecord struct {
	Id     string   `json:"id"`
	PostId string   `json:"postId"`
	Image  []string `json:"image"`
	Video  string   `json:"video"`
}

type PostRecord struct {
	Id                string   `json:"id"`
	Name              string   `json:"name"`
	UserId            string   `json:"userId"`
	AgentId           string   `json:"agentId"`
	Status            string   `json:"status"`
	State             string   `json:"state"`
	Province          string   `json:"province"`
	District          string   `json:"district"`
	Subdistrict       string   `json:"subdistrict"`
	LocationText      string   `json:"locationText"`
	Currency          string   `json:"currency"`
	Gender            string   `json:"gender"`
	Age               int      `json:"age"`
	Height            int      `json:"height"`
	Weight            int      `json:"weight"`
	ShapeChest        int      `json:"shapeChest"`
	ShapeWaist        int      `json:"shapeWaist"`
	ShapeAss          int      `json:"shapeAss"`
	Caption           string   `json:"caption"`
	LineId            string   `json:"lineId"`
	Telegram          string   `json:"telegram"`
	Description       string   `json:"description"`
	Location          Location `json:"location"`
	Option            any      `json:"option"`
	Taboo             any      `json:"taboo"`
	Tag               any      `json:"tag"`
	Image             []string `json:"image"`
	Video             string   `json:"video"`
	Verify            bool     `json:"verify"`
	Medical           any      `json:"medical"`
	Course            any      `json:"course"`
	ViewCount         int      `json:"viewCount"`
	UpdatedBy         string   `json:"updatedBy"`
	ExpireAt          string   `json:"expireAt"`
	IsSuperStar       bool     `json:"isSuperStar"`
	SuperStar         int      `json:"superStar"`
	SuperStarExpireAt string   `json:"superStarExpireAt"`
	Created           string   `json:"created"`
	Updated           string   `json:"updated"`
}

type Location struct {
	Lon float64 `json:"lon"`
	Lat float64 `json:"lat"`
}
