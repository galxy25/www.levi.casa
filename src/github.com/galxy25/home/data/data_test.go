package data

import (
	"testing"
)

var validConnectString = "74657374657240746573742e636f6d:sky 53616c75746174696f6e732c426f64792c4661726577656c6c false 1533515348\n"
var invalidConnectString = "invalid email:sky VGhpcyBpcyBhIG5ldyB0ZXN0ISDwn5Gp8J+Pu+KAjfCfkrs= false 1530410509\n"

var validDesiredConnection = Connection{
	Sender:                 "test@test@tester.com",
	Message:                "hi, so, bye",
	SubscribeToMailingList: true,
	ReceiveEpoch:           1531622217,
}

var validDesiredConnection2 = Connection{
	Sender:                 "white space",
	Message:                "hi 👶🏻, so, talk later",
	SubscribeToMailingList: true,
	ReceiveEpoch:           1531622217,
}

func TestConnectionFromStringSucceedsWithValidInput(t *testing.T) {
	_, err := ConnectionFromString(validConnectString)
	if err != nil {
		t.Errorf("failed to convert string %v to a connection: %v", validConnectString, err)
	}
}

func TestConnectionFromStringFailsIfEmailAddressIsInvalid(t *testing.T) {
	_, err := ConnectionFromString(invalidConnectString)
	if err == nil {
		t.Errorf("expected error when extracting connection from string %v", invalidConnectString)
	}
}

func TestConnectionToStringSucceedsWithValidInput(t *testing.T) {
	_ = validDesiredConnection.String()
	_ = validDesiredConnection2.String()
}
