package cqrs_test

import (
	"errors"
	"github.com/andrewwebber/cqrs"
	// "github.com/andrewwebber/cqrs/couchbase"
	"github.com/andrewwebber/cqrs/rethinkdb"
	r "github.com/dancannon/gorethink"
	"log"
	"testing"
)

type AccountCreatedEvent struct {
	FirstName    string
	LastName     string
	EmailAddress string
}

type EmailAddressChangedEvent struct {
	PreviousEmailAddress string
	NewEmailAddress      string
}

type Account struct {
	cqrs.EventSourceBased

	FirstName    string
	LastName     string
	EmailAddress string
}

func NewAccount(firstName string, lastName string, emailAddress string) *Account {
	account := new(Account)
	account.EventSourceBased = cqrs.NewEventSourceBased(account)

	event := AccountCreatedEvent{firstName, lastName, emailAddress}
	account.Update(&event)
	return account
}

func NewAccountFromHistory(id string, repository cqrs.EventSourcingRepository) (*Account, error) {
	account := new(Account)
	account.EventSourceBased = cqrs.NewEventSourceBasedWithID(account, id)

	if error := repository.Get(id, account); error != nil {
		return account, error
	}

	return account, nil
}

func (account *Account) ChangeEmailAddress(newEmailAddress string) error {
	if len(newEmailAddress) < 1 {
		return errors.New("Invalid newEmailAddress length")
	}

	account.Update(&EmailAddressChangedEvent{account.EmailAddress, newEmailAddress})
	return nil
}

func (account *Account) HandleAccountCreatedEvent(event *AccountCreatedEvent) {
	account.EmailAddress = event.EmailAddress
	account.FirstName = event.FirstName
	account.LastName = event.LastName
	log.Println("HandleAccountCreatedEvent ", event)
}

func (account *Account) HandleUsernameChangedEvent(event *EmailAddressChangedEvent) {
	account.EmailAddress = event.NewEmailAddress
	log.Println("HandleEmailAddressChangedEvent : ", event)
}

func TestEventSourcing(t *testing.T) {
	persistance := CreatePersistanceProvider(t)

	repository := cqrs.NewRepository(persistance)
	repository.RegisterAggregate(&Account{}, &AccountCreatedEvent{}, &EmailAddressChangedEvent{})
	accountID := "5058e029-d329-4c4b-b111-b042e48b0c5f"

	account := NewAccount("John", "Snow", "john.snow@cqrs.example")
	account.SetID(accountID)
	account.ChangeEmailAddress("john.snow@the.wall")

	log.Println(account.EmailAddress)
	repository.Save(account)

	account, error := NewAccountFromHistory(accountID, repository)
	if error != nil {
		t.Fatal(error)
	}

	log.Println(account.EmailAddress)

	account.ChangeEmailAddress("john.snow@golang.org")
	log.Println(account.EmailAddress)
	repository.Save(account)

	account, error = NewAccountFromHistory(accountID, repository)
	if error != nil {
		t.Fatal(error)
	}

	log.Println(account.EmailAddress)
}

func CreatePersistanceProvider(t *testing.T) cqrs.EventStreamRepository {
	connectOps := r.ConnectOpts{Address: "localhost:28015", Database: "cqrs"}
	session, error := r.Connect(connectOps)
	r.Table("events").Delete().Run(session)

	persistance, error := rethinkdb.NewRepository(connectOps, "events")
	if error != nil {
		t.Fatal(error)
	}

	r.Table("events").Delete().Run(session)

	return persistance
}
