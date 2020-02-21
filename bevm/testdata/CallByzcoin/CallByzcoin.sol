pragma solidity ^0.4.24;
pragma experimental ABIEncoderV2;

contract CallByzcoin {
    // Data structures

    struct Argument {
        string name;
        bytes value;
    }

    // Events

    event ByzcoinSpawn (
        bytes32 instanceID,
        string contractID,
        Argument[] args
    );

    event ByzcoinInvoke (
        bytes32 instanceID,
        string contractID,
        string command,
        Argument[] args
    );

    event ByzcoinDelete (
        bytes32 instanceID,
        Argument[] args
    );

    // Fields

    bytes32 instanceID;
    string contractID;

    // Constructor

    constructor(bytes32 _instanceID, string _contractID) public {
        instanceID = _instanceID;
        contractID = _contractID;
    }

    // Public functions

    function spawnValue(uint8 value) public {
        bytes memory argValue = new bytes(1);
        argValue[0] = byte(value);

        Argument[] memory args = new Argument[](1);
        args[0] = Argument({
            name: "value",
            value: argValue
        });

        emit ByzcoinSpawn(instanceID, contractID, args);
    }

    // Private functions
}

