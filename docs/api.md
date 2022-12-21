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
  * [ModelRenewOrder](#ModelRenewOrder)
  * [ModelShowCommits](#ModelShowCommits)
  * [ModelUpdate](#ModelUpdate)
  * [ModelUpdatePermission](#ModelUpdatePermission)
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
    "ID": "CovLVG4fQcqVT6KVTFJ4imsRN6dscKVYzoF6oqBkfMaxgPJvpFiRUyvz85Pv62LuCnNj92z",
    "Addrs": [
      "/ip4/127.0.0.1/tcp/26660",
      "/ip4/172.16.0.11/tcp/26660"
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
  "PeerInfo": "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT"
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
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "keyword": "fd248a7c-cf9f-4902-8327-58629aef96e9",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "keywordType": 1,
      "lastValidHeight": 711397,
      "gateway": "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT"
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  {
    "Proposal": {
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "provider": "cosmos197vlml2yg75rg9dmf07sau0mn0053p9dscrfsf",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "duration": 31536000,
      "replica": 1,
      "timeout": 86400,
      "alias": "notes",
      "dataId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
      "commitId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
      "cid": "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u",
      "size": 40,
      "operation": 1
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  42,
  "Ynl0ZSBhcnJheQ=="
]
```

Response:
```json
{
  "DataId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
  "Alias": "notes",
  "TxId": "",
  "Cid": "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u"
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
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "keyword": "fd248a7c-cf9f-4902-8327-58629aef96e9",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "keywordType": 1,
      "lastValidHeight": 711397,
      "gateway": "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT"
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  {
    "Proposal": {
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "provider": "cosmos197vlml2yg75rg9dmf07sau0mn0053p9dscrfsf",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "duration": 31536000,
      "replica": 1,
      "timeout": 86400,
      "alias": "notes",
      "dataId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
      "commitId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
      "cid": "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u",
      "size": 40,
      "operation": 1
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  42
]
```

Response:
```json
{
  "DataId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
  "Alias": "notes",
  "TxId": "",
  "Cid": "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u"
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
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "dataId": "fd248a7c-cf9f-4902-8327-58629aef96e9"
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  true
]
```

Response:
```json
{
  "DataId": "fd248a7c-cf9f-4902-8327-58629aef96e9",
  "Alias": "note_ca0b1124-f013-4c69-8249-41694d540871"
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
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "keyword": "fd248a7c-cf9f-4902-8327-58629aef96e9",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "keywordType": 1,
      "lastValidHeight": 711397,
      "gateway": "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT"
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  }
]
```

Response:
```json
{
  "DataId": "fd248a7c-cf9f-4902-8327-58629aef96e9",
  "Alias": "note_ca0b1124-f013-4c69-8249-41694d540871",
  "CommitId": "fd248a7c-cf9f-4902-8327-58629aef96e9",
  "Version": "v0",
  "Cid": "bafkreide7eax3pd3qsbolguprfta7thinb4wmbvyh2kestrdeiydg77tsq",
  "Content": "{\"content\":\"\",\"isEdit\":false,\"time\":\"2022-12-20 06:41\",\"title\":\"sample\"}"
}
```

### ModelRenewOrder
ModelRenewOrder renew a list of orders


Perms: write

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "duration": 42,
      "timeout": 32,
      "data": [
        "string value"
      ]
    },
    "JwsSignature": {
      "protected": "string value",
      "signature": "string value"
    }
  },
  true
]
```

Response:
```json
{
  "Results": {
    "1e05407f-a7af-4b1c-b9e5-99d492f07720": "New Order=1",
    "1e05407f-a7af-4b1c-b9e5-99d492f07721": "renew fail root cause"
  }
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
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "keyword": "fd248a7c-cf9f-4902-8327-58629aef96e9",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "keywordType": 1,
      "lastValidHeight": 711397,
      "gateway": "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT"
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  }
]
```

Response:
```json
{
  "DataId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
  "Alias": "notes",
  "Commits": [
    "c2b37317-9612-41fe-8260-7c8aea0dbd07\u001a711196",
    "85de5f5e-0cfb-4e0c-abe7-bf93aec087f3\u001a712565"
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
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "keyword": "fd248a7c-cf9f-4902-8327-58629aef96e9",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "keywordType": 1,
      "lastValidHeight": 711397,
      "gateway": "/ip4/172.16.0.10/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/tcp/26660/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/172.16.0.10/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT,/ip4/127.0.0.1/udp/26662/quic/webtransport/certhash/uEiCzHFKwct72TeBBh7-LUQ8L9QWwAo0b7d4VvsatjsQlQQ/certhash/uEiBKclz2BT5PNmQ9LIZr0DdhY7MpLLNXz8xLVdzSGyVXbA/p2p/12D3KooWR9jc8uHQ7T1n8Um5kt48usmNZxZftBKKEq9o4MYdFizT"
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  {
    "Proposal": {
      "owner": "did:sid:67a2be7315740823ebb6a27e2cfd7825fc02102a942235dd2589af47a2dafba4",
      "provider": "cosmos197vlml2yg75rg9dmf07sau0mn0053p9dscrfsf",
      "groupId": "30293f0f-3e0f-4b3c-aff1-890a2fdf063b",
      "duration": 31536000,
      "replica": 1,
      "timeout": 86400,
      "alias": "notes",
      "dataId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
      "commitId": "c2b37317-9612-41fe-8260-7c8aea0dbd07",
      "cid": "bafkreib3yoebpagjbkvhrsyhi7jpllylcqt4zpime5vho6ehpljv3dda4u",
      "size": 40,
      "operation": 1
    },
    "JwsSignature": {
      "protected": "eyJraWQiOiJkaWQ6c2lkOjY3YTJiZTczMTU3NDA4MjNlYmI2YTI3ZTJjZmQ3ODI1ZmMwMjEwMmE5NDIyMzVkZDI1ODlhZjQ3YTJkYWZiYTQ_dmVyc2lvbi1pZD02N2EyYmU3MzE1NzQwODIzZWJiNmEyN2UyY2ZkNzgyNWZjMDIxMDJhOTQyMjM1ZGQyNTg5YWY0N2EyZGFmYmE0IzhNalI1RlpCUUUiLCJhbGciOiJFUzI1NksifQ",
      "signature": "qbkzpCz_Yd8IeYmtmpGG2gdj-fkr5GwrHp5liBAOCSF5MQpHrZDFxp_GfTHv1sh8oDmR8JF2g9-GyVct7UJ24w"
    }
  },
  42,
  "Ynl0ZSBhcnJheQ=="
]
```

Response:
```json
{
  "DataId": "fd248a7c-cf9f-4902-8327-58629aef96e9",
  "CommitId": "fd248a7c-cf9f-4902-8327-58629aef96e9",
  "Alias": "notes",
  "TxId": "",
  "Cid": "bafkreide7eax3pd3qsbolguprfta7thinb4wmbvyh2kestrdeiydg77tsq"
}
```

### ModelUpdatePermission
ModelUpdatePermission update an existing model's read/write permission


Perms: write

Inputs:
```json
[
  {
    "Proposal": {
      "owner": "string value",
      "dataId": "string value",
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
  true
]
```

Response:
```json
{
  "DataId": "string value"
}
```

