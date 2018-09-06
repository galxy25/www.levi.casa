package communicator

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/galxy25/home/data"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strings"
)

const (
	defaultEmailEncoding = "UTF-8"                                      //Character encoding to use for email string fields.
	twilioBaseEndpoint   = "https://api.twilio.com/2010-04-01/Accounts" //Root URL for twilio related calls.
	maxSmsLength         = 1600                                         //https://www.twilio.com/docs/glossary/what-sms-character-limit
)

// Email address to use when email sender
// address is invalid/not specified.
var defaultReplyTo = fmt.Sprintf("%v@levi.casa", data.AnonToken)

// Which AWS SES regional endpoint to use for sending email.
var sesRegion = os.Getenv("HOME_SES_REGION")

// Which address to use as the origin for the
// email, must be verified with AWS SES.
var sesSource = os.Getenv("HOME_SES_SOURCE")

// Account ID for Twilio API calls.
var twilioSID = os.Getenv("TWILIO_ACCOUNT_SID")

// Auth token for Twilio API calls.
var twilioAuthToken = os.Getenv("TWILIO_AUTH_TOKEN")

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
		toAddress := aws.StringValue(&receiver)
		destinations.ToAddresses = append(destinations.ToAddresses, aws.String(toAddress))
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

// SMS represents a message that can be
// sent to and from PSTN endpoints
type SMS struct {
	Sender   string
	Receiver string
	Message  string
}

// Send sends an sms, returning error (if any).
func (s *SMS) Send() (err error) {
	err = smsPublisher(s)
	return err
}

// smsPublisher publishes an SMS message,
// returning error (if any).
// Stubbed during unit tests.
// https://www.twilio.com/blog/2014/06/sending-sms-from-your-go-app.html
var smsPublisher = func(sms *SMS) (err error) {
	messagesURI := fmt.Sprintf("%v/%v/Messages.json", twilioBaseEndpoint, twilioSID)
	urlParams := url.Values{}
	urlParams.Set("To", sms.Receiver)
	urlParams.Set("From", sms.Sender)
	urlParams.Set("Body", sms.Message)
	requestBody := *strings.NewReader(urlParams.Encode())
	request, err := http.NewRequest("POST", messagesURI, &requestBody)
	if err != nil {
		return err
	}
	request.SetBasicAuth(twilioSID, twilioAuthToken)
	request.Header.Add("Accept", "application/json")
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{}
	response, err := client.Do(request)
	var data map[string]interface{}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		return err
	}
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		packageLogger.WithFields(log.Fields{
			"executor": "#smsPublisher",
			"request":  urlParams,
			"response": data,
		}).Info("sms publish successful")
	} else {
		packageLogger.WithFields(log.Fields{
			"executor":      "#smsPublisher",
			"request":       urlParams,
			"response":      data,
			"response_code": response.StatusCode,
		}).Error("sms publish failed")
		errorMessage, ok := data["message"].(string)
		if !ok {
			errorMessage = "smsPublisher: unexpected twilio api response format"
		}
		err = errors.New(errorMessage)
	}
	return err
}
