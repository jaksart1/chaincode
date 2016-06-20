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
	"strings"
	"time"
	

	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Chainchat Chaincode implementation
type ChainchatChaincode struct {
}

var msgIndexStr = "_msgindex"				//name for the key/value that will store a list of all known messages
var openTradesStr = "_opentrades"			// no idea what this is

type Message struct{  // Message that gets added to blockchain
	MessageID 			string `json: "messageid"`
    Message 			string `json: "message"`
	// SenderName		string `json: "sendername"`
	// ReceiverName		string `json: "receivername"`
    SenderPublicKey 	string `json: "senderpublickey"`
	Time				string `json: "time"`
}

// ============================================================================================================================
// Main
// ============================================================================================================================
func main() {
    
    err := shim.Start(new(ChainchatChaincode))
    if err != nil {
        fmt.Printf("Error starting Chainchat chaincode: %s", err)
    }
}
// ============================================================================================================================
// Init - reset all the things
// ============================================================================================================================
func (t *ChainchatChaincode) Init(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	if function == "init" {
		err := stub.CreateTable("Receiver_Publickey", []*shim.ColumnDefinition{
			{"MessageID", shim.ColumnDefinition_STRING, false},
			{"Message", shim.ColumnDefinition_STRING, false},
			{"SenderPublicKey", shim.ColumnDefinition_STRING, false},
			{"Time", shim.ColumnDefinition_STRING, false},
		})

		if err != nil {
			fmt.Printf("Error creating table: %s", err)
		}
	}

	return nil, nil
}

// ============================================================================================================================
// Invoke - Our entry point for Invocations
// ============================================================================================================================
func (t *ChainchatChaincode) Invoke(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("invoke is running " + function)

	// Handle different functions
	if function == "init" {													//initialize the chaincode state, used as reset
		return t.Init(stub, "init", args)
	} else if function == "msg_delete" {									//deletes a message from its state
		return t.msg_delete(stub, args)
	} else if function == "msg_init" {										//creates a table with all message details
		return t.msg_init(stub, args)
	} 
	fmt.Println("invoke did not find func: " + function)					//error

	return nil, errors.New("Received unknown function invocation")
}

// msg_delete: Deletes messages and its associated table from the chainstate
func (t *SimpleChaincode) msg_delete(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
	
	public_key := args[0]
	
	err := stub.DeleteTable("Receiver_PublicKey")									//removes message associated table from chain state

	if err != nil {
		fmt.Printf("Error deleting table: %s", err)
	}

	return nil, nil
}
/* Marbles Delete function can be useful if we want to delete specific messages
// func (t *ChainchatChaincode) Delete(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
// 	if len(args) != 1 {
// 		return nil, errors.New("Incorrect number of arguments. Expecting 1")
// 	}
	
// 	name := args[0]
// 	err := stub.DelState(name)													//remove the key from chaincode state
// 	if err != nil {
// 		return nil, errors.New("Failed to delete state")
// 	}

// 	//get the marble index
// 	msgAsBytes, err := stub.GetState(chatIndexStr)
// 	if err != nil {
// 		return nil, errors.New("Failed to get chat index")
// 	}
// 	var chatIndex []string
// 	json.Unmarshal(msgAsBytes, &chatIndex)								//un stringify it aka JSON.parse()
	
// 	//remove marble from index
// 	for i,val := range chatIndex{
// 		fmt.Println(strconv.Itoa(i) + " - looking at " + val + " for " + name)
// 		if val == name{															//find the correct marble
// 			fmt.Println("found marble")
// 			chatIndex = append(chatIndex[:i], chatIndex[i+1:]...)			//remove it
// 			for x:= range chatIndex{											//debug prints...
// 				fmt.Println(string(x) + " - " + chatIndex[x])
// 			}
// 			break
// 		}
// 	}
// 	jsonAsBytes, _ := json.Marshal(chatIndex)									//save new index
// 	err = stub.PutState(chatIndexStr, jsonAsBytes)
// 	return nil, nil
// } */

// msg_init: Stores a message and its associated table in the chainstate
func (t *SimpleChaincode) msg_init(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	// args[0] - Message
	// args[1] - SenderPublicKey
	// args[2] - Time

	var msg_id int
	msg_id := 0 			//MessageID

	table, t_err := stub.GetTable("Receiver_PublicKey")
	if t_err == nil {
		rowAdded, r_err := stub.InsertRow(table.Name, shim.Row{
			[]*shim.Column{
				{&shim.Column_String_{msg_id++}},
				{&shim.Column_String_{time.Now().Format(time.RFC3339)}},
				{&shim.Column_String_{time.Now().Add(time.Hour * 24 * 365 * time.Duration(length)).Format(time.RFC3339)}},
				{&shim.Column_String_{time.Now().Format(time.RFC3339)}},
				{&shim.Column_Bool{true}},
				{&shim.Column_String_{""}},
				{&shim.Column_String_{args[1]}},
				{&shim.Column_String_{args[2]}},
			},
		})

		if r_err != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", r_err))
		}



}

func (t *SimpleChaincode) msg_unread(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	table, err := stub.GetTable("Receiver_PublicKey")
	if err != nil {
		return errors.New("The receiver public key message table does not exist")
	}

	row, r_err := stub.GetRows(table.Name, []shim.Column{})
	if r_err != nil || len(row.Columns) == 0 {
		return errors.New("Message records do not exists")
	}

	msg_delete(stub, args[0]) 

	return Message{
		MessageID:     			t.readStringSafe(row.Columns[0]),
		Message:     			t.readStringSafe(row.Columns[1]),
		SenderPublicKey:     	t.readStringSafe(row.Columns[2]),
		Time:     				t.readStringSafe(row.Columns[3]),
	}, nil 
}

func (t *SimpleChaincode) readStringSafe(col *shim.Column) string {
	if col == nil {
		return ""
	}

	return col.GetString_()
}

func (t *SimpleChaincode) readInt32Safe(col *shim.Column) int32 {
	if col == nil {
		return 0
	}

	return col.GetInt32()
}

func (t *SimpleChaincode) readBoolSafe(col *shim.Column) bool {
	if col == nil {
		return false
	}

	return col.GetBool()
}

// ============================================================================================================================
// Query - Used for reading data from the chainstate
// ============================================================================================================================
func (t *ChainchatChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "msg_unread" {											//retrieves unread messages from chaincode state
		return t.msg_unread(stub, args)
	}
	else if function == "msg_history" {	
																			//retrieves history of messages from blocks
	}
	fmt.Println("query did not find func: " + function)						//error
	return nil, errors.New("Received unknown function query")
}

//Read
func (t *ChainchatChaincode) read(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var , jsonResp string
	var err error

	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting name of the var to query")
	}

	name = args[0]
	valAsbytes, err := stub.GetState(name)									//get the var from chaincode state
	if err != nil {
		jsonResp = "{\"Error\":\"Failed to get state for " + name + "\"}"
		return nil, errors.New(jsonResp)
	}

	return valAsbytes, nil													//send it onward
}


//Delete


// //Write
// func (t *ChainchatChaincode) Write(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
// 	var name, value string // Entities
// 	var err error
// 	fmt.Println("running write()")

// 	if len(args) != 2 {
// 		return nil, errors.New("Incorrect number of arguments. Expecting 2. name of the variable and value to set")
// 	}

// 	name = args[0]															//rename for funsies
// 	value = args[1]
// 	err = stub.PutState(name, []byte(value))								//write the variable into the chaincode state
// 	if err != nil {
// 		return nil, err
// 	}
// 	return nil, nil
// }

//Init message
func (t *ChainchatChaincode) init_msg(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	var err error

	//   0       1       2    
	// "asdf", "You",   "Me"
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 4")
	}

	fmt.Println("- start init message")
	if len(args[0]) <= 0 {
		return nil, errors.New("1st argument must be a non-empty string")
	}
	if len(args[1]) <= 0 {
		return nil, errors.New("2nd argument must be a non-empty string")
	}
	if len(args[2]) <= 0 {
		return nil, errors.New("3rd argument must be a non-empty string")
	}


	to := strings.ToLower(args[1])
	user := strings.ToLower(args[2])

	str := `{"msg": "` + args[0] + `", "To": "` + to + `, "user": "` + user + `"}`
	err = stub.PutState(args[0], []byte(str))								//store message with id as key
	if err != nil {
		return nil, err
	}
		
	//get the chat index
	msgAsBytes, err := stub.GetState(chatIndexStr)
	if err != nil {
		return nil, errors.New("Failed to get marble index")
	}
	var chatIndex []string
	json.Unmarshal(msgAsBytes, &chatIndex)							//un stringify it aka JSON.parse()
	
	//append
	chatIndex = append(chatIndex, args[0])								//add marble name to index list
	fmt.Println("! chat index: ", chatIndex)
	jsonAsBytes, _ := json.Marshal(chatIndex)
	err = stub.PutState(chatIndexStr, jsonAsBytes)						//store name of marble

	fmt.Println("- end init marble")
	return nil, nil
}
