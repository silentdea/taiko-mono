package server

import (
	"bytes"
	"encoding/hex"
	"log"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/labstack/echo/v4"
	"github.com/taikoxyz/taiko-mono/packages/taiko-client/pkg/rpc"
)

// @title Taiko Proposer Server API
// @version 1.0
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://community.taiko.xyz/
// @contact.email info@taiko.xyz

// @license.name MIT
// @license.url https://github.com/taikoxyz/taiko-mono/packages/taiko-client/blob/main/LICENSE.md

type buildBlockRequest struct {
	L1StateBlockNumber uint32   `json:"l1StateBlockNumber"`
	Timestamp          uint64   `json:"timestamp"`
	SignedTransactions []string `json:"signedTransactions"`
	Coinbase           string   `json:"coinbase"`
	ExtraData          string   `json:"extraData"`
}

type buildBlockResponse struct {
	RLPEncodedTx string `json:"rlpEncodedTx"`
}

// BuildBlock handles a query to build a block according to our protocol, given the inputs,
// and returns an unsigned transaction to `taikol1.ProposeBlock`.
//
//	@Summary		Build a block and return an unsigned `taikol1.ProposeBlock` transaction
//	@ID			   	build
//	@Accept			json
//	@Produce		json
//	@Success		200	{object} BuildBlockResponse
//	@Router			/block/build [get]
func (s *PreconfAPIServer) BuildBlock(c echo.Context) error {
	req := &buildBlockRequest{}
	if err := c.Bind(req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, err)
	}

	var transactions types.Transactions

	for _, signedTxHex := range req.SignedTransactions {
		if strings.HasPrefix(signedTxHex, "0x") {
			signedTxHex = signedTxHex[2:]
		}

		rlpEncodedBytes, err := hex.DecodeString(signedTxHex)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, err)
		}

		var tx types.Transaction
		if err := rlp.DecodeBytes(rlpEncodedBytes, &tx); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, err)
		}

		transactions = append(transactions, &tx)
	}

	txListBytes, err := rlp.EncodeToBytes(transactions)
	if err != nil {
		log.Fatalf("Failed to RLP encode transactions: %v", err)
	}

	tx, err := s.txBuilder.BuildUnsigned(
		c.Request().Context(),
		txListBytes,
		req.L1StateBlockNumber,
		req.Timestamp,
		common.HexToAddress(req.Coinbase),
		rpc.StringToBytes32(req.ExtraData),
	)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, err)
	}

	// RLP encode the transaction
	var rlpEncodedTx bytes.Buffer
	if err := rlp.Encode(&rlpEncodedTx, tx); err != nil {
		log.Fatalf("Failed to RLP encode the transaction: %v", err)
	}

	hexEncodedTx := hex.EncodeToString(rlpEncodedTx.Bytes())

	return c.JSON(http.StatusOK, buildBlockResponse{RLPEncodedTx: hexEncodedTx})
}