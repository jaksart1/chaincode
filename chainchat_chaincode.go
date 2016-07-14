/*
Copyright IBM Corp 2016 All Rights Reserved.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements.  See the NOTICE file
distributed with this work for additional information
regarding copyright ownership.  The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License.  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied.  See the License for the
specific language governing permissions and limitations
under the License.
*/

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// ChainchatChaincode implementation
type ChainchatChaincode struct {
}

// MessageResults stores an array of retrieved messages
type MessageResults struct {
	Messages []Message `json:"messages"`
}

// Message stores a message and its related info
type Message struct { // Message that gets added to blockchain
	MessageID       int64  `json:"messageid"`
	Message         string `json:"message"`
	SenderPublicKey string `json:"senderpublickey"`
	Time            string `json:"time"`
}

// Main initializes the shim
func main() {
	err := shim.Start(new(ChainchatChaincode))
	if err != nil {
		fmt.Printf("Error starting Chainchat chaincode: %s", err)
	}
}

// Init creates the messages table and starts the ID counter to 0
func (t *ChainchatChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {

	if function == "init" {
		// Create the table where all messages will be stored
		err := stub.CreateTable("messages", []*shim.ColumnDefinition{
			{"ReceiverPublicKey", shim.ColumnDefinition_STRING, true},
			{"MessageID", shim.ColumnDefinition_UINT64, true},
			{"Message", shim.ColumnDefinition_STRING, false},
			{"SenderPublicKey", shim.ColumnDefinition_STRING, false},
			{"Time", shim.ColumnDefinition_STRING, false},
		})

		// Handle table creation errors
		if err != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not create messages table: %s", err))
			return nil, err
		}

		// Initialize the message counter to 0
		err = stub.PutState("counter", []byte("0"))

		// Handle counter state initialization errors
		if err != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not initialize message counter to 0: %s", err))
			return nil, err
		}
	}

	return nil, nil
}

// Invoke is the entry point for invocations, and handles initializing and read messages from the ledger
func (t *ChainchatChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	// Handle the different invocations

	// Initialize the chaincode state, used as reset
	if function == "init" {
		return t.Init(stub, "init", args)
	} else if function == "deleteMsg" { // Deletes a message from the ledger
		return t.deleteMsg(stub, args)
	} else if function == "writeMsg" { // Add a message to the ledger
		return t.writeMsg(stub, args)
	}

	// The requested invoke function was not recognized
	fmt.Println(fmt.Sprintf("[ERROR] Did not recognize the function to invoke: %s", function))

	return nil, errors.New("Received unknown function invocation")
}

// deleteMsg accepts a message ID and deletes it from the ledger
func (t *ChainchatChaincode) deleteMsg(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	err := stub.DeleteTable("Receiver_PublicKey") //removes message associated table from chain state

	if err != nil {
		fmt.Printf("Error deleting table: %s", err)
	}

	return nil, nil
}

// writeMsg stores a new message on the ledger
// args[0] - Message
// args[1] - SenderPublicKey
// args[2] - Time
// args[3] - Receiver_PublicKey
func (t *ChainchatChaincode) writeMsg(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var msgID uint64      // The message's unique ID
	msg := args[0]        // The message itself
	sendPubKey := args[1] // The sender's public key
	recvPubKey := args[3] // The receiver's public key
	timestamp := args[2]  // The message's timestamp

	// Check that we have the write number of args
	if len(args) != 4 {
		fmt.Println(fmt.Sprintf("[ERROR] Expected 4 arguments but got %v", len(args)))
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	// Retrieve the message ID and parse it as a number
	byteID, err := stub.GetState("counter")
	if err != nil {
		fmt.Println(fmt.Sprintf("[ERROR] Could not retrieve the message's ID: %s", err))
		return nil, err
	}
	strID := string(byteID[:])
	msgID, err = strconv.ParseUint(strID, 10, 64)
	if err != nil {
		fmt.Println(fmt.Sprintf("[ERROR] Could not parse the message ID to a number: %s", err))
		return nil, err
	}

	fmt.Println(fmt.Sprintf("[INFO] Receiver public key: " + recvPubKey))

	// Attempt to add a row for the message
	rowAdded, rowErr := stub.InsertRow("messages", shim.Row{
		Columns: []*shim.Column{
			{&shim.Column_String_{String_: recvPubKey}},
			{&shim.Column_Uint64{Uint64: msgID}},
			{&shim.Column_String_{String_: msg}},
			{&shim.Column_String_{String_: sendPubKey}},
			{&shim.Column_String_{String_: timestamp}},
		},
	})

	if rowErr != nil || !rowAdded {
		fmt.Println(fmt.Sprintf("[ERROR] Could not insert a message into the ledger: %s", rowErr))
		return nil, rowErr
	}

	// Increment the message ID and store the updated one
	msgID++
	stub.PutState("counter", []byte(strconv.FormatUint(msgID, 10)))
	return nil, nil
}

// readMsgs accepts a receiver's public key and returns all the messages destined for that receiver
func (t *ChainchatChaincode) readMsgs(stub *shim.ChaincodeStub, args []string) ([]Message, error) {
	// The public key we're getting messages for
	recvPubKey := args[0]

	// Error out if not enough arguments
	if len(args) != 1 {
		fmt.Println(fmt.Sprintf("[ERROR] Expected one argument but got: %v", len(args)))
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	// Retrieve all the rows that are messages for the specified user
	rows, rowErr := stub.GetRows("messages", []shim.Column{shim.Column{Value: &shim.Column_String_{String_: recvPubKey}}})
	if rowErr != nil {
		fmt.Println(fmt.Sprintf("[ERROR] Could not retrieve the rows: %s", rowErr))
		return nil, rowErr
	}

	// Parse the results into a message array
	var messageResults []Message

	fmt.Println(fmt.Sprintf("[INFO] Receiver public key: " + recvPubKey))
	fmt.Println(len(rows))

	for index := 0; index < len(rows); index++ {
		var currentRow = <-rows
		strMsgID := t.readStringSafe(currentRow.Columns[0])

		messageResults = append(messageResults, Message{
			MessageID:       t.readInt64Safe(currentRow.Columns[0]),
			Message:         t.readStringSafe(currentRow.Columns[1]),
			SenderPublicKey: t.readStringSafe(currentRow.Columns[2]),
			Time:            t.readStringSafe(currentRow.Columns[3]),
		})

		deleteErr := stub.DeleteRow("messages", []shim.Column{{&shim.Column_String_{String_: strMsgID}}})

		if deleteErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] Could not delete message row for ID %s: %s", strMsgID, deleteErr))
		}
	}

	return messageResults, nil
}

func (t *ChainchatChaincode) readStringSafe(col *shim.Column) string {
	if col == nil {
		return ""
	}

	return col.GetString_()
}

func (t *ChainchatChaincode) readInt64Safe(col *shim.Column) int64 {
	if col == nil {
		return 0
	}

	return col.GetInt64()
}

func (t *ChainchatChaincode) readBoolSafe(col *shim.Column) bool {
	if col == nil {
		return false
	}

	return col.GetBool()
}

// Query queries the chainstate
func (t *ChainchatChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	if function == "readMsgs" { // Retrieves all messages for a particular user
		messageResults, readErr := t.readMsgs(stub, args)

		// Handle retrieval errors
		if readErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] There was an error in reading the messages: %s", readErr.Error()))
			return nil, readErr
		}

		// Marshal the messages into a JSON string
		converted, marshalErr := json.Marshal(MessageResults{messageResults})

		// Handle marshalling errors
		if marshalErr != nil {
			fmt.Println(fmt.Sprintf("[ERROR] There was a marshalling error: %s", marshalErr.Error()))
			return nil, marshalErr
		}

		fmt.Println(converted)

		return converted, nil
	}

	fmt.Println("[ERROR] Query did not find func " + function) //error
	return nil, errors.New("Received unknown function query")
}
