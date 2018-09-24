package data

import (
	"encoding/hex"
	"fmt"
	"testing"
)

var validConnectString = "74657374657240746573742e636f6d 736b79 1530410509 53616c75746174696f6e732c426f64792c4661726577656c6c 1533515348\n"
var validConnectString2 = "74657374657240746573742e636f6d 736b79 1530410509 53616c75746174696f6e732c426f64792c4661726577656c6c\n"
var invalidConnectString = "invalid email:sky VGhpcyBpcyBhIG5ldyB0ZXN0ISDwn5Gp8J+Pu+KAjfCfkrs= false 1530410509\n"
var invalidConnectString2 = "74657374657240746573742e636f6d 736b79 1533515348 53616c75746174696f6e732c426f64792c4661726577656c6c false\n"
var invalidConnectString3 = "74657374657240746573742e636f6d\n"
var validDesiredConnection = Connection{
	Sender:       "test@test@tester.com",
	Receiver:     "sky@levi.casa",
	Message:      "hi, so, bye",
	SendEpoch:    1531622217,
	ReceiveEpoch: 1535403864,
}
var validDesiredConnection2 = Connection{
	Sender:       "+14155552671",
	Receiver:     "+15038002120",
	Message:      "S & M & S",
	SendEpoch:    1531622217,
	ReceiveEpoch: 1535403864,
}
var validDesiredConnection3 = Connection{
	Sender:    "white space",
	Message:   "hi 👶🏻, so, talk later",
	SendEpoch: 1531622217,
}

func TestConnectionFromStringSucceedsWithValidInput(t *testing.T) {
	validStrings := []string{
		validConnectString,
		validConnectString2,
	}
	for _, validString := range validStrings {
		_, err := ConnectionFromString(validString)
		if err != nil {
			t.Errorf("failed to convert string %v to a connection: %v", validString, err)
		}
	}
}

func TestConnectionFromStringFailsIfEmailAddressIsInvalid(t *testing.T) {
	invalidStrings := []string{
		invalidConnectString,
		invalidConnectString2,
		invalidConnectString3,
	}
	for _, invalidString := range invalidStrings {
		_, err := ConnectionFromString(invalidString)
		if err == nil {
			t.Errorf("expected error when extracting connection from string %v", invalidConnectString)
		}
	}
}

func TestConnectionToStringSucceedsWithValidInput(t *testing.T) {
	connectString := validDesiredConnection.String()
	expectedString := fmt.Sprintf("%v %v %v %v %v", hex.EncodeToString([]byte(validDesiredConnection.Sender)), hex.EncodeToString([]byte(validDesiredConnection.Receiver)), validDesiredConnection.SendEpoch, hex.EncodeToString([]byte(validDesiredConnection.Message)), validDesiredConnection.ReceiveEpoch)
	if connectString != expectedString {
		t.Errorf("expected %v, got %v\n", expectedString, connectString)
	}
	_ = validDesiredConnection2.String()
	connectString = validDesiredConnection3.String()
	expectedString = fmt.Sprintf("%v %v %v %v %v", hex.EncodeToString([]byte(validDesiredConnection3.Sender)), hex.EncodeToString([]byte(validDesiredConnection3.Receiver)), validDesiredConnection3.SendEpoch, hex.EncodeToString([]byte(validDesiredConnection3.Message)), validDesiredConnection3.ReceiveEpoch)
	if connectString != expectedString {
		t.Errorf("expected %v, got %v\n", expectedString, connectString)
	}
}
