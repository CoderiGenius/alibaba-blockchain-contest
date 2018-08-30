// +build !experimental

/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"encoding/json"
	"fmt"
	//"strconv"
	//"time"


	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/hyperledger/fabric/core/chaincode/shim/ext/entities"
	"github.com/hyperledger/fabric/bccsp/factory"
	//"github.com/pkg/errors"
	"regexp"
)

const DECKEY = "DECKEY"
const VERKEY = "VERKEY"
const ENCKEY = "ENCKEY"
const SIGKEY = "SIGKEY"
const IV = "IV"

// EncCC example simple Chaincode implementation of a chaincode that uses encryption/signatures
type EncCC struct {
	bccspInst bccsp.BCCSP
}
type Input struct {

	YearValue string
	SchoolOrCompanyValue string
	IdentityValue string
}

func getStateAndDecrypt(stub shim.ChaincodeStubInterface, ent entities.Encrypter, key string,year string) ([]byte, error) {
	// at first we retrieve the ciphertext from the ledger
	ciphertext, err := stub.GetHistoryForKey(key)
	if err != nil {
		return nil, err
	}

	defer ciphertext.Close()


	var keys []string
	for ciphertext.HasNext() {
		response, iterErr := ciphertext.Next()
		if iterErr != nil {
			return nil,err
		}
		var returnObject Input
		//var decObject Input
		fmt.Println("undecValue:"+string(response.Value))
		decObject,_ := ent.Decrypt(response.Value)
		json.Unmarshal(decObject,&returnObject)
		fmt.Println(returnObject.YearValue)
		if(returnObject.YearValue == year){
			fmt.Println("query:"+returnObject.SchoolOrCompanyValue)
			returnJson,_ := json.Marshal(returnObject.SchoolOrCompanyValue)
			//returnJson := returnObject.SchoolOrCompanyValue
			return returnJson,err
		}
		keys = append(keys, string(response.Value))
	}

	for key, txID := range keys {
		fmt.Printf("key %d contains %s\n", key, txID)
	}

	return nil,err

}

// encryptAndPutState encrypts the supplied value using the
// supplied entity and puts it to the ledger associated to
// the supplied KVS key
func encryptAndPutState(stub shim.ChaincodeStubInterface, ent entities.Encrypter, key string, value []byte) error {
	// at first we use the supplied entity to encrypt the value
	ciphertext, err := ent.Encrypt(value)
	if err != nil {
		return err
	}

	return stub.PutState(key, ciphertext)
}

// getStateDecryptAndVerify retrieves the value associated to key,
// decrypts it with the supplied entity, verifies the signature
// over it and returns the result of the decryption in case of
// success
func getStateDecryptAndVerify(stub shim.ChaincodeStubInterface, ent entities.EncrypterSignerEntity, key string) ([]byte, error) {
	// here we retrieve and decrypt the state associated to key
	val, err := getStateAndDecrypt(stub, ent, key,"")
	if err != nil {
		return nil, err
	}

	// we unmarshal a SignedMessage from the decrypted state
	msg := &entities.SignedMessage{}
	err = msg.FromBytes(val)
	if err != nil {
		return nil, err
	}

	// we verify the signature
	ok, err := msg.Verify(ent)
	if err != nil {
		return nil, err
	} else if !ok {
		return nil, nil
	}

	return msg.Payload, nil
}

// signEncryptAndPutState signs the supplied value, encrypts
// the supplied value together with its signature using the
// supplied entity and puts it to the ledger associated to
// the supplied KVS key
func signEncryptAndPutState(stub shim.ChaincodeStubInterface, ent entities.EncrypterSignerEntity, key string, value []byte) error {
	// here we create a SignedMessage, set its payload
	// to value and the ID of the entity and
	// sign it with the entity
	msg := &entities.SignedMessage{Payload: value, ID: []byte(ent.ID())}
	err := msg.Sign(ent)
	if err != nil {
		return err
	}

	// here we serialize the SignedMessage
	b, err := msg.ToBytes()
	if err != nil {
		return err
	}

	// here we encrypt the serialized version associated to args[0]
	return encryptAndPutState(stub, ent, key, b)
}

type keyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// getStateByRangeAndDecrypt retrieves a range of KVS pairs from the
// ledger and decrypts each value with the supplied entity; it returns
// a json-marshalled slice of keyValuePair
func getStateByRangeAndDecrypt(stub shim.ChaincodeStubInterface, ent entities.Encrypter, startKey, endKey string) ([]byte, error) {
	// we call get state by range to go through the entire range
	iterator, err := stub.GetStateByRange(startKey, endKey)
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	// we decrypt each entry - the assumption is that they have all been encrypted with the same key
	keyvalueset := []keyValuePair{}
	for iterator.HasNext() {
		el, err := iterator.Next()
		if err != nil {
			return nil, err
		}

		v, err := ent.Decrypt(el.Value)
		if err != nil {
			return nil, err
		}

		keyvalueset = append(keyvalueset, keyValuePair{el.Key, string(v)})
	}

	bytes, err := json.Marshal(keyvalueset)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}


// Encrypter exposes how to write state to the ledger after having
// encrypted it with an AES 256 bit key that has been provided to the chaincode through the
// transient field
func (t *EncCC) Encrypter(stub shim.ChaincodeStubInterface, args []string, encKey, IV []byte) []byte {
	// create the encrypter entity - we give it an ID, the bccsp instance, the key and (optionally) the IV
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, encKey, IV)
	if err != nil {
		return []byte("entities.NewAES256EncrypterEntity failed, err %s")
	}



	key := args[0]

	encryptGroup := Input{
		YearValue :   args[1] ,
		SchoolOrCompanyValue : args[2] ,
		IdentityValue : args[3],
	}
	cleartextValue,_ := json.Marshal(encryptGroup)

	// here, we encrypt cleartextValue and assign it to key
	err = encryptAndPutState(stub, ent, key, cleartextValue)
	if err != nil {
		return nil
	}
	return nil
}

// Decrypter exposes how to read from the ledger and decrypt using an AES 256
// bit key that has been provided to the chaincode through the transient field.
func (t *EncCC) Decrypter(stub shim.ChaincodeStubInterface, args []string, decKey, IV []byte) []byte{
	// create the encrypter entity - we give it an ID, the bccsp instance, the key and (optionally) the IV
	ent, err := entities.NewAES256EncrypterEntity("ID", t.bccspInst, decKey, IV)
	if err != nil {
		return []byte("entities.NewAES256EncrypterEntity failed, err %s")
	}



	key := args[0]
	year := args[1]

	// here we decrypt the state associated to key
	cleartextValue, err := getStateAndDecrypt(stub, ent, key,year)
	if err != nil {
		return []byte("getStateAndDecrypt failed, err %+v")
	}
	fmt.Println("cleartextValue:"+string(cleartextValue))

	reg := regexp.MustCompile("\"")
	temp := []byte("")

	//返回str中第一个匹配reg的字符串
	data := reg.ReplaceAll(cleartextValue,temp)


	// here we return the decrypted value as a result
	return data
}

// EncrypterSigner exposes how to write state to the ledger after having received keys for
// encrypting (AES 256 bit key) and signing (X9.62/SECG curve over a 256 bit prime field) that has been provided to the chaincode through the
// transient field




// This chaincode implements a simple map that is stored in the state.
// The following operations are available.

// Invoke operations
// put - requires two arguments, a key and value
// remove - requires a key
// get - requires one argument, a key, and returns a value
// keys - requires no arguments, returns all keys

// SimpleChaincode example simple Chaincode implementation
// EncCC example simple Chaincode implementation of a chaincode that uses encryption/signatures




// Init does nothing for this cc
func (t *EncCC) Init(stub shim.ChaincodeStubInterface) pb.Response {

	return shim.Success(nil)
}
// Invoke has two functions
// put - takes two arguments, a key and value, and stores them in the state
// remove - takes one argument, a key, and removes if from the state
func (t *EncCC) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	tMap, _ := stub.GetTransient()
	switch function {

	case "addRecord":
		fmt.Println("addRecord")
		if len(args) < 4 {
			return shim.Error("put operation must include four arguments")
		}
		//定义结构体变量
		//var inputOject input
		//获取参数
		//inputOject.IDKey = args[0]
		/*inputOject.yearValue = args[1]
		inputOject.schoolOrCompanyValue = args[2]
		inputOject.identityValue = args[3]
*/
		group := Input {
			//IDKey :     args[0] ,
			YearValue :   args[1] ,
			SchoolOrCompanyValue : args[2] ,
			IdentityValue : args[3],
		}



		//输出结果
		fmt.Println("arg0"+args[0])
		fmt.Println("arg1"+args[1])
		fmt.Println("arg2"+args[2])
		fmt.Println("arg3"+args[3])
		//序列化
		Value,err := json.Marshal(group)
		fmt.Println(string(Value))
		fmt.Println(Value)
		if err = stub.PutState(args[0], Value); err != nil {
				fmt.Printf("Error putting state %s", err)
			return shim.Error("put operation failed. Error updating state: %s")
		}

		indexName := args[0]
		compositeKey, err := stub.CreateCompositeKey(indexName, []string{args[1]})
		fmt.Println(compositeKey)
		if err != nil {
			return shim.Error(err.Error())
		}

		valueByte := Value
		if err := stub.PutState(compositeKey, valueByte); err != nil {
			fmt.Printf("Error putting state with compositeKey %s", err)
			return shim.Error("put operation failed. Error updating state with compositeKey: %s")
		}

		return shim.Success(nil)





	case "getRecord":
		key := args[0]
		keysIter, err := stub.GetHistoryForKey(key)
		if err != nil {
			return shim.Error("query operation failed. Error accessing state: %s")
		}
		defer keysIter.Close()

		var keys []string
		for keysIter.HasNext() {
			response, iterErr := keysIter.Next()
			if iterErr != nil {
				return shim.Error("query operation failed. Error accessing state: %s")
			}
			var returnObject Input
			json.Unmarshal(response.Value,&returnObject)
			fmt.Println(returnObject.YearValue)
			if(returnObject.YearValue == args[1]){
				fmt.Println("query:"+returnObject.SchoolOrCompanyValue)
				returnJson,_ := json.Marshal(returnObject.SchoolOrCompanyValue)

				reg := regexp.MustCompile("\"")
				temp := []byte("")

				//返回str中第一个匹配reg的字符串
				data2 := reg.ReplaceAll(returnJson,temp)


				return shim.Success(data2)
			}
			keys = append(keys, string(response.Value))
		}

		for key, txID := range keys {
			fmt.Printf("key %d contains %s\n", key, txID)
		}

		//jsonKeys, err := json.Marshal(keys)
		if err != nil {
			return shim.Error("query operation failed. Error marshaling JSON: %s")
		}

		return shim.Error("")



	case "encRecord":
		// make sure there's a key in transient - the assumption is that
		// it's associated to the string "ENCKEY"
		if _, in := tMap[ENCKEY]; !in {
			return shim.Error("Expected transient encryption key %s")
		}

		return shim.Success(t.Encrypter(stub, args[0:], tMap[ENCKEY], tMap[IV]))
	case "decRecord":

		// make sure there's a key in transient - the assumption is that
		// it's associated to the string "DECKEY"
		if _, in := tMap[DECKEY]; !in {
			return shim.Error("Expected transient decryption key %s")
		}

		returnByte := t.Decrypter(stub, args[0:], tMap[DECKEY], tMap[IV])

		reg := regexp.MustCompile("\"")
		temp := []byte("")

		//返回str中第一个匹配reg的字符串
		data3 := reg.ReplaceAll(returnByte,temp)

		return shim.Success(data3)

	default:
		return shim.Error("Unsupported operation")
	}
}


func main() {
	factory.InitFactories(nil)
	err := shim.Start(&EncCC{factory.GetDefault()})
	if err != nil {
		fmt.Printf("Error starting chaincode: %s", err)
	}
}
