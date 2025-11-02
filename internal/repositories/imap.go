package repositories

import (
	"app/internal/domains"
	"log"
	"strings"
	"time"

	"github.com/BrianLeishman/go-imap"
	"github.com/patrickmn/go-cache"
)

type imapRepository struct {
	client  *imap.Dialer
	mailbox string
}

type ImapRepository interface {
	FetchEmailsFromSenderWithSubject(senderFilter, subjectFilter string,old uint) ([]string, error)
	ListenForNewEmails(senderFilter, subjectFilter string) (domains.Listening, domains.StopListening, chan string, chan error)
}

var mapEmail = cache.New(time.Hour, time.Hour)

func NewImapRepository(server string, port int, username, password, mailbox string) (ImapRepository, error) {
	// imap.Verbose = true

	imap.RetryCount = 3

	imap.TLSSkipVerify = true

	im, err := imap.New(username, password, server, port)
	if err != nil {
		return nil, err
	}

	err = im.SelectFolder("Inbox")
	if err != nil {
		return nil, err
	}
	return &imapRepository{
		client:  im,
		mailbox: mailbox,
	}, nil
}

func (r *imapRepository) FetchEmailsFromSenderWithSubject(senderFilter, subjectFilter string, old uint) ([]string, error) {
	uids, err := r.client.GetUIDs("999999999:*")
	if err != nil {
		return nil, err
	}

	result := []string{}
	uidsList := []int{}
	for i := 0; i <= int(old); i++ {
		v := uids[0] - i
		if v < 1 {
			break
		}
		uidsList = append(uidsList, v)
	}

	emails, err := r.client.GetEmails(uidsList...)
	if err != nil {
		return nil, err
	}
	for _, v := range emails {
		if v != nil {
			if strings.Contains(v.Subject, subjectFilter) && strings.Contains(v.From.String(), senderFilter) {
				result = append(result, v.Text)
			}
		}
	}

	return result, nil
}

func (r *imapRepository) ListenForNewEmails(senderFilter, subjectFilter string) (domains.Listening, domains.StopListening, chan string, chan error) {
	errChan := make(chan error, 1)
	strMailBodyChan := make(chan string, 100)

	stopFunc := func() {
		r.client.StopIdle()
	}

	updateFunc := &imap.IdleHandler{
		OnExists: func(event imap.ExistsEvent) {
			emails, err := r.FetchEmailsFromSenderWithSubject(senderFilter, subjectFilter,10)
			if err != nil {
				errChan <- err
				return
			}
			for _, v := range emails {
				strMailBodyChan <- v
			}
		},
		OnExpunge: func(event imap.ExpungeEvent) {
			log.Printf("Email deleted: %d", event.MessageIndex)
		},
		OnFetch: func(event imap.FetchEvent) {
			log.Printf("Email fetched: UID %d", event.UID)
		},
	}

	startFunc := func() error {
		return r.client.StartIdle(updateFunc)
	}

	return startFunc, stopFunc, strMailBodyChan, errChan
}
