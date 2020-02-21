package bevm

import (
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/xerrors"
)

func eventByID(eventsAbi abi.ABI, eventID common.Hash) *abi.Event {
	for _, event := range eventsAbi.Events {
		if event.Id() == eventID {
			return &event
		}
	}

	return nil
}

func unpackEvent(contractAbi abi.ABI, eventID common.Hash,
	eventBytes []byte) (interface{}, error) {

	event := eventByID(contractAbi, eventID)
	if event == nil {
		// Event not found
		return nil, nil
	}

	eventInputs := event.Inputs

	switch len(eventInputs) {
	case 0:
		return nil, nil

	case 1:
		// Create a pointer to the desired type
		result := reflect.New(eventInputs[0].Type.Type)

		err := contractAbi.Unpack(result.Interface(),
			event.Name, eventBytes)
		if err != nil {
			return nil, xerrors.Errorf("failed to unpack single "+
				"result of EVM execution: %v", err) // FIXME
		}

		// Dereference the result pointer
		return result.Elem().Interface(), nil

	default:
		// Abi.Unpack() on multiple values supports a struct or array/slice as
		// structure into which the result is stored. Struct is cleaner, but it
		// does not support unnamed outputs ("or purely underscored"). If this
		// is needed, an array implementation, commented out, follows.

		// Build a struct naming the fields after the outputs
		var fields []reflect.StructField
		for _, output := range eventInputs {
			// Adapt names to what Abi.Unpack() does
			name := abi.ToCamelCase(output.Name)

			fields = append(fields, reflect.StructField{
				Name: name,
				Type: output.Type.Type,
			})
		}

		structType := reflect.StructOf(fields)
		s := reflect.New(structType)

		err := contractAbi.Unpack(s.Interface(),
			event.Name, eventBytes)
		if err != nil {
			return nil, xerrors.Errorf("failed to unpack multiple "+
				"result of EVM execution: %v", err) // FIXME
		}

		// Dereference the result pointer
		return s.Elem().Interface(), nil

		// // Build an array of interface{}
		// var empty interface{}
		// arrType := reflect.ArrayOf(len(abiOutputs),
		// 	reflect.ValueOf(&empty).Type().Elem())
		// result := reflect.New(arrType)

		// // Create a value of the desired type for each output
		// for i, output := range abiOutputs {
		// 	val := reflect.New(output.Type.Type)
		// 	result.Elem().Index(i).Set(val)
		// }

		// err := contractInstance.Parent.Abi.Unpack(result.Interface(),
		// 	method, resultBytes)
		// if err != nil {
		// 	return nil, xerrors.Errorf("unpacking multiple result: %v", err)
		// }

		// for i := range abiOutputs {
		// 	val := result.Elem().Index(i)
		// 	// Need to dereference values twice:
		// 	// val is interface{}, *val is *type, **val is type
		// 	val.Set(val.Elem().Elem())
		// }

		// // Dereference the result pointer
		// return result.Elem().Interface(), nil
	}
}

const eventsAbiJSON = `` +
	`[` +
	`  {` +
	`    "anonymous": false,` +
	`    "inputs": [` +
	`      {` +
	`        "indexed": false,` +
	`        "name": "instanceID",` +
	`        "type": "bytes32"` +
	`      },` +
	`      {` +
	`        "indexed": false,` +
	`        "name": "contractID",` +
	`        "type": "string"` +
	`      },` +
	`      {` +
	`        "components": [` +
	`          {` +
	`            "name": "name",` +
	`            "type": "string"` +
	`          },` +
	`          {` +
	`            "name": "value",` +
	`            "type": "bytes"` +
	`          }` +
	`        ],` +
	`        "indexed": false,` +
	`        "name": "args",` +
	`        "type": "tuple[]"` +
	`      }` +
	`    ],` +
	`    "name": "ByzcoinSpawn",` +
	`    "type": "event"` +
	`  },` +
	`  {` +
	`    "anonymous": false,` +
	`    "inputs": [` +
	`      {` +
	`        "indexed": false,` +
	`        "name": "instanceID",` +
	`        "type": "bytes32"` +
	`      },` +
	`      {` +
	`        "indexed": false,` +
	`        "name": "contractID",` +
	`        "type": "string"` +
	`      },` +
	`      {` +
	`        "indexed": false,` +
	`        "name": "command",` +
	`        "type": "string"` +
	`      },` +
	`      {` +
	`        "components": [` +
	`          {` +
	`            "name": "name",` +
	`            "type": "string"` +
	`          },` +
	`          {` +
	`            "name": "value",` +
	`            "type": "bytes"` +
	`          }` +
	`        ],` +
	`        "indexed": false,` +
	`        "name": "args",` +
	`        "type": "tuple[]"` +
	`      }` +
	`    ],` +
	`    "name": "ByzcoinInvoke",` +
	`    "type": "event"` +
	`  },` +
	`  {` +
	`    "anonymous": false,` +
	`    "inputs": [` +
	`      {` +
	`        "indexed": false,` +
	`        "name": "instanceID",` +
	`        "type": "bytes32"` +
	`      },` +
	`      {` +
	`        "components": [` +
	`          {` +
	`            "name": "name",` +
	`            "type": "string"` +
	`          },` +
	`          {` +
	`            "name": "value",` +
	`            "type": "bytes"` +
	`          }` +
	`        ],` +
	`        "indexed": false,` +
	`        "name": "args",` +
	`        "type": "tuple[]"` +
	`      }` +
	`    ],` +
	`    "name": "ByzcoinDelete",` +
	`    "type": "event"` +
	`  }` +
	`]`
