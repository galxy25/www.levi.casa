package communicator

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"net/mail"
	"os"
)

const (
	defaultEmailEncoding = "UTF-8" //Character encoding to use for email string fields.
)

// Email address to use when email sender
// address is invalid/not specified.
var defaultReplyTo = fmt.Sprintf("%v@levi.casa", data.AnonToker)

// Which AWS SES regional endpoint to use for sending email.
var sesRegion = os.Getenv("HOME_SES_REGION")

// Which address to use as the origin for the
// email, must be verified with AWS SES.
var sesSource = os.Getenv("HOME_SES_SOURCE")

// Email represents a message that can be
// sent using the SMPT protocol.
// https://tools.ietf.org/html/rfc5321
type Email struct {
	Subject   string
	Message   string
	Sender    string
	Receivers []string
}

// Send sends an email,
// returning error (if any).
func (e *Email) Send() (err error) {
	response, err := sesPublisher(e)
	packageLogger.WithFields(log.Fields{
		"executor": "Email.#Send",
		"email":    e,
		"response": response,
	}).Info(response)
	return err
}

// sesPublisher sends email using AWS SES
// returning response and error (if any).
// Stubbed during unit tests.
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/ses-example-send-email.html
var sesPublisher = func(email *Email) (response *ses.SendEmailOutput, err error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(sesRegion)},
	)
	svc := ses.New(sess)
	// Assemble the email.
	destinations := &ses.Destination{}
	for _, receiver := range email.Receivers {
		destinations.ToAddresses = append(destinations.ToAddresses, aws.String(receiver))
	}
	// This way users can semi-anonymously send
	// an email(though they won't be able to receive a reply).
	// Semi-anonymously because sender's IP may or may not be in a server log(I'm not keeping it, but user's can't and shouldn't trust that!).
	// XXX: future semi-anonymous message sends will generate a burner link for continuing communications.
	emailAddress, invalidEmailAddress := mail.ParseAddress(email.Sender)
	var replyTo string
	if invalidEmailAddress != nil {
		replyTo = defaultReplyTo
	} else {
		replyTo = emailAddress.String()
	}
	input := &ses.SendEmailInput{
		Destination: destinations,
		Source:      aws.String(sesSource),
		// Source above is the origin, and
		// must be verified with SES.
		// ReplyToAddresses can be anybody.
		ReplyToAddresses: []*string{&replyTo},
		Message: &ses.Message{
			Body: &ses.Body{
				Text: &ses.Content{
					Charset: aws.String(defaultEmailEncoding),
					Data:    aws.String(email.Message),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(defaultEmailEncoding),
				Data:    aws.String(email.Subject),
			},
		},
	}
	// Attempt to send the email.
	response, err = svc.SendEmail(input)
	return response, err
}
