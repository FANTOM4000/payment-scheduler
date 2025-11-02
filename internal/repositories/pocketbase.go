package repositories

import (
	"app/internal/domains"
	"fmt"
	"io"
	"log"

	"github.com/robfig/cron"
	"resty.dev/v3"
)

type pocketBase struct {
	address   string
	isReady   bool
	client    *resty.Client
	cronjob   *cron.Cron
	superuser domains.SuperUserRecord
}

type PocketBase interface {
	Subscribe(collection string, T domains.PaymentRecord) (domains.Listening, domains.StopListening, chan domains.RecordHook[domains.PaymentRecord], chan error)
	IsReady() bool
	Close()
	UpdateRecord(collection string, id string, record map[string]any) error
	GetPaymentRecordByFilter(collection string, filter string) ([]domains.PaymentRecord, error)
	GetPaymentRecordById(collection string, id string, T domains.PaymentRecord) (domains.PaymentRecord, error)
	CreateRecord(collection string, record map[string]any) (domains.CreateRecordResponse, error)
	GetSuperUser() domains.SuperUserRecord
	GetRequestTimeForFreePostFilter(collection string, filter string) ([]domains.RequestTimeForFreePostRecord, error)
	DeleteRecord(collection string, id string) error
	GetPostById(collection string, id string) (domains.PostRecord, error)
	 GetPostRecordByFilter(collection string, filter string) ([]domains.PostRecord, error)
	GetFileFromObjectKey(collection string, id string, objectKey string) (io.Reader, error)
	AddFile(collection string, id string, fieldName string, file ...io.Reader) error
}

func NewPocketBase(address, username, password string) PocketBase {
	client := resty.New()
	client.SetBaseURL(address)
	authResponse := domains.AuthResponse{}
	resp, err := client.R().
		SetBody(map[string]any{
			"identity": username,
			"password": password,
		}).
		SetResult(&authResponse).
		Post("/api/collections/_superusers/auth-with-password")
	if err != nil {
		panic(err)
	}
	if resp.IsError() {
		panic(resp.Error())
	}
	if authResponse.Token == "" {
		panic("empty auth token")
	}
	client.SetAuthToken(authResponse.Token)
	cronjob := cron.New()
	p := &pocketBase{
		address:   address,
		isReady:   true,
		client:    client,
		cronjob:   cronjob,
		superuser: authResponse.Record,
	}
	cronjob.AddFunc("@hourly", func() {
		reAuthResponse := domains.AuthResponse{}
		reAuthResp, err := p.client.R().SetResult(&reAuthResponse).Post("/api/collections/_superusers/auth-refresh")
		if err != nil {
			log.Print("Error re-authenticating:", err)
			p.isReady = false
			return
		}
		if reAuthResp.IsError() {
			log.Print("Error re-authenticating:", reAuthResp.Error())
			p.isReady = false
			return
		}
		if reAuthResponse.Token == "" {
			log.Print("Empty token on re-authentication")
			p.isReady = false
			return
		}
		p.client.SetAuthToken(reAuthResponse.Token)
		p.superuser = reAuthResponse.Record
		log.Print("Re-authenticated successfully")
	})
	cronjob.Start()
	return p
}

type CallbackFunc[T any] func(domains.RecordHook[T]) error

func (p *pocketBase) Subscribe(collection string, T domains.PaymentRecord) (domains.Listening, domains.StopListening, chan domains.RecordHook[domains.PaymentRecord], chan error) {
	errChan := make(chan error, 1)
	recordChan := make(chan domains.RecordHook[domains.PaymentRecord], 1)
	eventSource := resty.NewEventSource()
	eventSource.OnOpen(func(url string) {
		fmt.Println("I'm connected:", url)
	})
	eventSource.SetURL(p.address+"api/realtime").
		OnError(func(err error) {
			fmt.Println("Error occurred:", err)
		}).
		AddHeader("Authorization", p.client.AuthToken()).
		AddEventListener(collection, func(a any) {
			evt := a.(*domains.RecordHook[domains.PaymentRecord])
			log.Printf("Received event for collection %s: %+v", collection, evt)
			recordChan <- *evt
		}, domains.RecordHook[domains.PaymentRecord]{}).
		AddEventListener("PB_CONNECT", func(a any) {
			evt := a.(*resty.Event)
			resp, err := p.client.R().SetFormData(map[string]string{
				"clientId":      evt.ID,
				"subscriptions": collection,
			}).Post("api/realtime")
			if err != nil {
				log.Print("Error subscribing to collection:", err)
				errChan <- err
			}
			if resp.IsError() {
				log.Print("Error subscribing to collection:", resp.String())
				errChan <- fmt.Errorf("error subscribing to collection: %s", resp.String())
			}
			log.Printf("Subscribed to collection %s successfully status %d", collection, resp.StatusCode())
		}, nil)

	listeningFunc := func() error {
		return eventSource.Get()
	}

	stopListening := func() {
		eventSource.Close()
	}

	return listeningFunc, stopListening, recordChan, errChan
}

func (p *pocketBase) IsReady() bool {
	return p.isReady
}

func (p *pocketBase) Close() {
	p.isReady = false
}

func (p *pocketBase) UpdateRecord(collection string, id string, record map[string]any) error {
	_, err := p.client.R().
		SetBody(record).
		Patch("/api/collections/" + collection + "/records/" + id)
	return err
}

func (p *pocketBase) GetPaymentRecordByFilter(collection string, filter string) ([]domains.PaymentRecord, error) {
	response := domains.ListRecordsResponse[domains.PaymentRecord]{}
	resp, err := p.client.R().
		SetQueryParams(map[string]string{
			"filter": filter,
			"limit":  "1",
		}).
		SetResult(&response).
		Get("/api/collections/" + collection + "/records")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error fetching record: %s", resp.String())
	}
	if len(response.Items) == 0 {
		return nil, fmt.Errorf("record not found with filter: %s", filter)
	}
	return response.Items, nil
}
func (p *pocketBase) GetPaymentRecordById(collection string, id string, T domains.PaymentRecord) (domains.PaymentRecord, error) {
	response := domains.PaymentRecord(T)
	resp, err := p.client.R().
		SetResult(&response).
		Get("/api/collections/" + collection + "/records/" + id)
	if err != nil {
		return domains.PaymentRecord{}, err
	}
	if resp.IsError() {
		return domains.PaymentRecord{}, fmt.Errorf("error fetching record by id: %s", resp.String())
	}
	return response, nil
}

func (p *pocketBase) CreateRecord(collection string, record map[string]any) (domains.CreateRecordResponse, error) {
	response := domains.CreateRecordResponse{}
	resp, err := p.client.R().
		SetBody(record).
		SetResult(&response).
		Post("/api/collections/" + collection + "/records")
	if err != nil {
		return domains.CreateRecordResponse{}, err
	}
	if resp.IsError() {
		return domains.CreateRecordResponse{}, fmt.Errorf("error creating record: %s", resp.String())
	}
	return response, nil
}

func (p *pocketBase) GetSuperUser() domains.SuperUserRecord {
	return p.superuser
}

func (p *pocketBase) GetRequestTimeForFreePostFilter(collection string, filter string) ([]domains.RequestTimeForFreePostRecord, error) {
	response := domains.ListRecordsResponse[domains.RequestTimeForFreePostRecord]{}
	resp, err := p.client.R().
		SetQueryParams(map[string]string{
			"filter": filter,
			"limit":  "20",
		}).
		SetResult(&response).
		Get("/api/collections/" + collection + "/records")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error fetching record: %s", resp.String())
	}
	if len(response.Items) == 0 {
		return nil, fmt.Errorf("record not found with filter: %s", filter)
	}
	return response.Items, nil
}

func (p *pocketBase) DeleteRecord(collection string, id string) error {
	_, err := p.client.R().
		Delete("/api/collections/" + collection + "/records/" + id)
	return err
}

func (p *pocketBase) GetPostById(collection string, id string) (domains.PostRecord, error) {
	response := domains.PostRecord{}
	resp, err := p.client.R().
		SetQueryParams(map[string]string{
			"filter": fmt.Sprintf("id='%s'", id),
			"limit":  "1",
		}).
		SetResult(&response).
		Get("/api/collections/" + collection + "/records")
	if err != nil {
		return domains.PostRecord{}, err
	}
	if resp.IsError() {
		return domains.PostRecord{}, fmt.Errorf("error fetching record by id: %s", resp.String())
	}
	return response, nil
}

func (p *pocketBase) GetFileFromObjectKey(collection string, id string, objectKey string) (io.Reader, error) {
	resp, err := p.client.R().
		SetDoNotParseResponse(true).
		Get(fmt.Sprintf("/api/files/%s/%s/%s", collection, id, objectKey))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error fetching file from url: %s", resp.String())
	}
	return resp.Body, nil
}

func (p *pocketBase) AddFile(collection string, id string, fieldName string, file ...io.Reader) error {
	r := p.client.R()
	for _, f := range file {
		r.SetFileReader(fieldName, "file", f)
	}
	resp, err := r.Patch("/api/collections/" + collection + "/records/" + id)
	if err != nil {
		return err
	}
	if resp.IsError() {
		return fmt.Errorf("error updating file: %s", resp.String())
	}
	return nil
}

func (p *pocketBase) GetPostRecordByFilter(collection string, filter string) ([]domains.PostRecord, error) {
	response := domains.ListRecordsResponse[domains.PostRecord]{}
	resp, err := p.client.R().
		SetQueryParams(map[string]string{
			"filter": filter,
			"limit":  "20",
		}).
		SetResult(&response).
		Get("/api/collections/" + collection + "/records")
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("error fetching record: %s", resp.String())
	}
	if len(response.Items) == 0 {
		return nil, fmt.Errorf("record not found with filter: %s", filter)
	}
	return response.Items, nil
}
