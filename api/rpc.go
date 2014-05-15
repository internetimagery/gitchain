package api

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"net/http"

	"github.com/gitchain/gitchain/env"
	"github.com/gitchain/gitchain/keys"
	"github.com/gitchain/gitchain/server"
	"github.com/gitchain/gitchain/transaction"
	"github.com/gitchain/gitchain/types"

	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
)

func jsonRpcService() *rpc.Server {
	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")
	s.RegisterService(new(KeyService), "")
	s.RegisterService(new(NameService), "")
	s.RegisterService(new(BlockService), "")
	return s
}

// KeySerice
type KeyService struct{}

type ImportPrivateKeyArgs struct {
	Alias string
	PEM   []byte
}

type ImportPrivateKeyReply struct {
	Success bool
}

func (srv *KeyService) ImportPrivateKey(r *http.Request, args *ImportPrivateKeyArgs, reply *ImportPrivateKeyReply) error {
	err := env.DB.PutKey(args.Alias, args.PEM, false)
	if err != nil {
		reply.Success = false
		return err
	}
	reply.Success = true
	return nil
}

type ListPrivateKeysArgs struct {
}

type ListPrivateKeysReply struct {
	Aliases []string
}

func (srv *KeyService) ListPrivateKeys(r *http.Request, args *ListPrivateKeysArgs, reply *ListPrivateKeysReply) error {
	reply.Aliases = env.DB.ListKeys()
	return nil
}

type SetMainKeyArgs struct {
	Alias string
}

type SetMainKeyReply struct {
	Success bool
}

func (srv *KeyService) SetMainKey(r *http.Request, args *SetMainKeyArgs, reply *SetMainKeyReply) error {
	key := env.DB.GetKey(args.Alias)
	if key != nil {
		err := env.DB.PutKey(args.Alias, key, true)
		if err != nil {
			return err
		}
		reply.Success = true
	} else {
		reply.Success = false
	}
	return nil
}

type GetMainKeyArgs struct {
}

type GetMainKeyReply struct {
	Alias string
}

func (srv *KeyService) GetMainKey(r *http.Request, args *GetMainKeyArgs, reply *GetMainKeyReply) error {
	keys := env.DB.ListKeys()
	mainKey := env.DB.GetMainKey()
	for i := range keys {
		if bytes.Compare(mainKey, env.DB.GetKey(keys[i])) == 0 {
			reply.Alias = keys[i]
		}
	}
	return nil
}

// NameService
type NameService struct{}

type NameReservationArgs struct {
	Alias string
	Name  string
}

type NameReservationReply struct {
	Id     string
	Random string
}

func (srv *NameService) NameReservation(r *http.Request, args *NameReservationArgs, reply *NameReservationReply) error {
	pemBlock, err := keys.ReadPEM(env.DB.GetKey(args.Alias), false)
	if err != nil {
		return err
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(pemBlock.Bytes)
	if err != nil {
		return err
	}
	tx, random := transaction.NewNameReservation(args.Name, &privateKey.PublicKey)
	reply.Id = hex.EncodeToString(tx.Hash())
	reply.Random = hex.EncodeToString(random)
	server.BroadcastTransaction(tx)
	return nil
}

// BlockService
type BlockService struct{}

type GetLastBlockArgs struct {
}

type GetLastBlockReply struct {
	Hash string
}

func (srv *BlockService) GetLastBlock(r *http.Request, args *GetLastBlockArgs, reply *GetLastBlockReply) error {
	block, err := env.DB.GetLastBlock()
	if err != nil {
		return err
	}
	reply.Hash = hex.EncodeToString(block.Hash())
	return nil
}

type GetBlockArgs struct {
	Hash string
}

type GetBlockReply struct {
	PreviousBlockHash types.Hash
	MerkleRootHash    types.Hash
	Timestamp         int64
	Bits              uint32
	Nonce             uint32
	NumTransactions   int
}

func (srv *BlockService) GetBlock(r *http.Request, args *GetBlockArgs, reply *GetBlockReply) error {
	hash, err := hex.DecodeString(args.Hash)
	if err != nil {
		return err
	}
	block, err := env.DB.GetBlock(hash)
	if err != nil {
		return err
	}
	reply.PreviousBlockHash = block.PreviousBlockHash
	reply.MerkleRootHash = block.MerkleRootHash
	reply.Timestamp = block.Timestamp
	reply.Bits = block.Bits
	reply.Nonce = block.Nonce
	reply.NumTransactions = len(block.Transactions)
	return nil
}