# Groups
* [Auth](#Auth)
  * [AuthNew](#AuthNew)
  * [AuthVerify](#AuthVerify)
* [Common](#Common)
  * [GenerateToken](#GenerateToken)
  * [GetHttpUrl](#GetHttpUrl)
  * [GetIpfsUrl](#GetIpfsUrl)
  * [GetNetPeers](#GetNetPeers)
  * [GetNodeAddress](#GetNodeAddress)
  * [GetPeerInfo](#GetPeerInfo)
* [Model](#Model)
  * [ModelCreate](#ModelCreate)
  * [ModelCreateFile](#ModelCreateFile)
  * [ModelDelete](#ModelDelete)
  * [ModelLoad](#ModelLoad)
  * [ModelShowCommits](#ModelShowCommits)
  * [ModelUpdate](#ModelUpdate)
## Auth


### AuthNew


Perms: admin

Inputs:
```json
[
  [
    "write"
  ]
]
```

Response: `"Ynl0ZSBhcnJheQ=="`

### AuthVerify
There are not yet any comments for this method.

Perms: none

Inputs:
```json
[
  "string value"
]
```

Response:
```json
[
  "write"
]
```

## Common


### GenerateToken
GenerateToken


Perms: read

Inputs:
```json
[
  "string value"
]
```

Response:
```json
{
  "Server": "string value",
  "Token": "string value"
}
```

### GetHttpUrl
GetHttpUrl


Perms: read

Inputs:
```json
[
  "string value"
]
```

Response:
```json
{
  "Url": "string value"
}
```

### GetIpfsUrl
GetIpfsUrl


Perms: read

Inputs:
```json
[
  "string value"
]
```

Response:
```json
{
  "Url": "string value"
}
```

### GetNetPeers
GetNetPeers get current node's connected peer list


Perms: read

Inputs: `null`

Response:
```json
[
  {
    "ID": "5G3K37EdUF",
    "Addrs": [
      "string value"
    ]
  }
]
```

### GetNodeAddress
GetNodeAddress get current node's sao chain address


Perms: read

Inputs: `null`

Response: `"string value"`

### GetPeerInfo
GetPeerInfo get current node's peer information


Perms: read

Inputs: `null`

Response:
```json
{
  "PeerInfo": "string value"
}
```

## Model
The Model method group contains methods for manipulating data models.


### ModelCreate
ModelCreate create a normal data model


Perms: write

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "keyword": "string value",
      "groupId": "string value",
      "keywordType": 32,
      "lastValidHeight": 42,
      "gateway": "string value",
      "commitId": "string value",
      "version": "string value"
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  {
    "Proposal": {
      "owner": "string value",
      "provider": "string value",
      "groupId": "string value",
      "duration": 42,
      "replica": 32,
      "timeout": 32,
      "alias": "string value",
      "dataId": "string value",
      "commitId": "string value",
      "tags": [
        "string value"
      ],
      "cid": "string value",
      "rule": "string value",
      "extendInfo": "string value",
      "size": 42,
      "operation": 32,
      "readonlyDids": [
        "string value"
      ],
      "readwriteDids": [
        "string value"
      ]
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  42,
  "Ynl0ZSBhcnJheQ=="
]
```

Response:
```json
{
  "DataId": "string value",
  "Alias": "string value",
  "TxId": "string value",
  "Cid": "string value"
}
```

### ModelCreateFile
ModelCreateFile create data model as a file


Perms: write

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "keyword": "string value",
      "groupId": "string value",
      "keywordType": 32,
      "lastValidHeight": 42,
      "gateway": "string value",
      "commitId": "string value",
      "version": "string value"
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  {
    "Proposal": {
      "owner": "string value",
      "provider": "string value",
      "groupId": "string value",
      "duration": 42,
      "replica": 32,
      "timeout": 32,
      "alias": "string value",
      "dataId": "string value",
      "commitId": "string value",
      "tags": [
        "string value"
      ],
      "cid": "string value",
      "rule": "string value",
      "extendInfo": "string value",
      "size": 42,
      "operation": 32,
      "readonlyDids": [
        "string value"
      ],
      "readwriteDids": [
        "string value"
      ]
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  42
]
```

Response:
```json
{
  "DataId": "string value",
  "Alias": "string value",
  "TxId": "string value",
  "Cid": "string value"
}
```

### ModelDelete
ModelDelete delete an existing model


Perms: write

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "dataId": "string value"
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  }
]
```

Response:
```json
{
  "DataId": "string value",
  "Alias": "string value"
}
```

### ModelLoad
ModelLoad load an existing data model


Perms: read

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "keyword": "string value",
      "groupId": "string value",
      "keywordType": 32,
      "lastValidHeight": 42,
      "gateway": "string value",
      "commitId": "string value",
      "version": "string value"
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  }
]
```

Response:
```json
{
  "DataId": "string value",
  "Alias": "string value",
  "CommitId": "string value",
  "Version": "string value",
  "Cid": "string value",
  "Content": "string value"
}
```

### ModelShowCommits
ModelShowCommits list a data models' historical commits


Perms: read

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "keyword": "string value",
      "groupId": "string value",
      "keywordType": 32,
      "lastValidHeight": 42,
      "gateway": "string value",
      "commitId": "string value",
      "version": "string value"
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  }
]
```

Response:
```json
{
  "DataId": "string value",
  "Alias": "string value",
  "Commits": [
    "string value"
  ]
}
```

### ModelUpdate
ModelUpdate update an existing data model


Perms: write

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "keyword": "string value",
      "groupId": "string value",
      "keywordType": 32,
      "lastValidHeight": 42,
      "gateway": "string value",
      "commitId": "string value",
      "version": "string value"
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  {
    "Proposal": {
      "owner": "string value",
      "provider": "string value",
      "groupId": "string value",
      "duration": 42,
      "replica": 32,
      "timeout": 32,
      "alias": "string value",
      "dataId": "string value",
      "commitId": "string value",
      "tags": [
        "string value"
      ],
      "cid": "string value",
      "rule": "string value",
      "extendInfo": "string value",
      "size": 42,
      "operation": 32,
      "readonlyDids": [
        "string value"
      ],
      "readwriteDids": [
        "string value"
      ]
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  42,
  "Ynl0ZSBhcnJheQ=="
]
```

Response:
```json
{
  "DataId": "string value",
  "CommitId": "string value",
  "Alias": "string value",
  "TxId": "string value",
  "Cid": "string value"
}
```

