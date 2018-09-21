package communicator

import (
	"errors"
	"fmt"
	"github.com/galxy25/home/data"
	"os"
	"strings"
)

var (
	ErrorNoContent                = errors.New("communicator/translate: message must have content")
	ErrorSmsMessageLengthExceeded = errors.New(fmt.Sprintf("communicator/translate: sms message size must be less than %v characters", maxSmsLength))
	ErrorUnknownConnectionType    = errors.New("communicator/translate: unknown connection type")
)

// Phone number for congressional telephonic communications.
var congressPhone = os.Getenv("CONGRESS_PHONE_NUMBER")

// Translate attempts to translate a raw connection into a
// send-able connection, returning error (if any).
// Translation occurs using the following algorithm
// keyed off the connection receiver field:
//   if receiver has an '@' character, translate to email
//   else if receiver begins with '+' character, translate to sms
//   else "translate" to nil Sender
func Translate(connection *data.Connection) (Sender, error) {
	receiver := connection.Receiver
	emailSigilPresent := strings.ContainsAny(receiver, "@")
	if emailSigilPresent {
		return EmailFromConnection(connection)
	}
	smsSigilPresent := strings.HasPrefix(receiver, "+")
	if smsSigilPresent {
		return SmsFromConnection(connection)
	}
	return nil, ErrorUnknownConnectionType
}

func EmailFromConnection(connection *data.Connection) (*Email, error) {
	if len(connection.Message) == 0 {
		return nil, ErrorNoContent
	}
	email := Email{
		Message:   connection.Message,
		Sender:    connection.Sender,
		Receivers: []string{connection.Receiver},
		Subject:   fmt.Sprintf("%v -> www.levi.casa", connection.Sender),
	}
	return &email, nil
}

func SmsFromConnection(connection *data.Connection) (*SMS, error) {
	messageLength := len(connection.Message)
	if messageLength == 0 {
		return nil, ErrorNoContent
	}
	messagePrefix := fmt.Sprintf("From: %v", connection.Sender)
	prefixLength := len(messagePrefix)
	if messageLength+prefixLength > maxSmsLength {
		return nil, ErrorSmsMessageLengthExceeded
	}
	sms := SMS{
		Message:  fmt.Sprintf("%v\n %v", messagePrefix, connection.Message),
		Sender:   congressPhone,
		Receiver: connection.Receiver,
	}
	return &sms, nil
}
