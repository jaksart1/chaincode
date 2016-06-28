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
	//"strconv"
	//"strings"
	//"time"
	"github.com/hyperledger/fabric/core/chaincode/shim"
)

// Chainchat Chaincode implementation
type ChainchatChaincode struct {
}

var msgIndexStr = "_msgindex"				//name for the key/value that will store a list of all known messages
var openTradesStr = "_opentrades"			// no idea what this is

type MessageResults struct {
	Messages []Message `json:"messages"`
}

type Message struct {  // Message that gets added to blockchain
	MessageID 			int64 	`json:"messageid"`
    Message 			string 	`json:"message"`
    SenderPublicKey 	string 	`json:"senderpublickey"`
	Time				string 	`json:"time"`
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
	if function == "init" {
		err := stub.CreateTable("Receiver_Publickey", []*shim.ColumnDefinition{
			{"MessageID", shim.ColumnDefinition_INT64, false},
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
func (t *ChainchatChaincode) msg_delete(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}
	
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
func (t *ChainchatChaincode) msg_init(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
	if len(args) != 3 {
		return nil, errors.New("Incorrect number of arguments. Expecting 3")
	}

	// args[0] - Message
	// args[1] - SenderPublicKey
	// args[2] - Time

	var msg_id int = 0											//MessageID

	table, t_err := stub.GetTable("Receiver_PublicKey")

	if t_err == nil {
		
		msg_id++												//increments MessageID for every new message received 

		rowAdded, r_err := stub.InsertRow(table.Name, shim.Row{
			[]*shim.Column{
				{&shim.Column_Int64{int64(msg_id)}},
				{&shim.Column_String_{args[0]}},				 
				{&shim.Column_String_{args[1]}},
				{&shim.Column_String_{args[2]}},
			},
		})

		if r_err != nil || !rowAdded {
			return nil, errors.New(fmt.Sprintf("Error creating row: %s", r_err))
		}
	}
	return nil,nil
}

func (t *ChainchatChaincode) msg_unread(stub *shim.ChaincodeStub, args []string) ([]Message, error) {
	if len(args) != 1 {
		return nil, errors.New("Incorrect number of arguments. Expecting 1")
	}

	table, err := stub.GetTable("Receiver_PublicKey")
	if err != nil {
		return nil, errors.New("The receiver public key message table does not exist")
	}

	rows, r_err := stub.GetRows(table.Name, []shim.Column{})
	if r_err != nil || len(rows) == 0 {
		return nil, errors.New("Message records do not exists")
	}

	// msg_delete(stub, args[0])							// not sure if we need to call delete here

	var messageResults []Message

	for index:=0; index < len(rows); index++ {
		var currentRow shim.Row = <- rows

		messageResults = append(messageResults, Message{
		MessageID:     			t.readInt64Safe(currentRow.Columns[0]),
		Message:     			t.readStringSafe(currentRow.Columns[1]),
		SenderPublicKey:     	t.readStringSafe(currentRow.Columns[2]),
		Time:     				t.readStringSafe(currentRow.Columns[3]),
	})
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

// ============================================================================================================================
// Query - Used for reading data from the chainstate
// ============================================================================================================================
func (t *ChainchatChaincode) Query(stub *shim.ChaincodeStub, function string, args []string) ([]byte, error) {
	fmt.Println("query is running " + function)

	// Handle different functions
	if function == "msg_unread" {											//retrieves unread messages from chaincode state
	messageResults, messageResults_err := t.msg_unread(stub, args)
	if messageResults_err != nil {
		return nil, errors.New("{\"error\":\"" + messageResults_err.Error() + "\"}")
	}
	converted, converted_err := json.Marshal(MessageResults{messageResults})
	if converted_err != nil {
		return nil, errors.New("{\"error\":\"" + converted_err.Error() + "\"}")
	}
	return converted, nil
	}
	// else if function == "msg_history" {	
	// 	return t.msg_history(stub, args)									//retrieves history of messages from blocks														
	// }
	fmt.Println("query did not find func: " + function)						//error
	return nil, errors.New("Received unknown function query")
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

// //Init message
// func (t *ChainchatChaincode) init_msg(stub *shim.ChaincodeStub, args []string) ([]byte, error) {
// 	var err error

// 	//   0       1       2    
// 	// "asdf", "You",   "Me"
// 	if len(args) != 3 {
// 		return nil, errors.New("Incorrect number of arguments. Expecting 4")
// 	}

// 	fmt.Println("- start init message")
// 	if len(args[0]) <= 0 {
// 		return nil, errors.New("1st argument must be a non-empty string")
// 	}
// 	if len(args[1]) <= 0 {
// 		return nil, errors.New("2nd argument must be a non-empty string")
// 	}
// 	if len(args[2]) <= 0 {
// 		return nil, errors.New("3rd argument must be a non-empty string")
// 	}


// 	to := strings.ToLower(args[1])
// 	user := strings.ToLower(args[2])

// 	str := `{"msg": "` + args[0] + `", "To": "` + to + `, "user": "` + user + `"}`
// 	err = stub.PutState(args[0], []byte(str))								//store message with id as key
// 	if err != nil {
// 		return nil, err
// 	}
		
// 	//get the chat index
// 	msgAsBytes, err := stub.GetState(chatIndexStr)
// 	if err != nil {
// 		return nil, errors.New("Failed to get marble index")
// 	}
// 	var chatIndex []string
// 	json.Unmarshal(msgAsBytes, &chatIndex)							//un stringify it aka JSON.parse()
	
// 	//append
// 	chatIndex = append(chatIndex, args[0])								//add marble name to index list
// 	fmt.Println("! chat index: ", chatIndex)
// 	jsonAsBytes, _ := json.Marshal(chatIndex)
// 	err = stub.PutState(chatIndexStr, jsonAsBytes)						//store name of marble

// 	fmt.Println("- end init marble")
// 	return nil, nil
// }
