package bevm

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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
	eventBytes []byte) (string, interface{}, error) {

	event := eventByID(contractAbi, eventID)
	if event == nil {
		// Event not found
		return "", nil, nil
	}

	eventData, err := unpackData(contractAbi, event.Name, eventBytes,
		event.Inputs)

	return event.Name, eventData, err
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
