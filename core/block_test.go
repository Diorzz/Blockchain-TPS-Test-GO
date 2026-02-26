package core

import (
	"reflect"
	"testing"

	"github.com/izqui/helpers"
)

func TestMerkellHash(t *testing.T) {

	tr1 := NewTransaction(nil, nil, []byte(helpers.RandomString(helpers.RandomInt(0, 1024*1024))))
	tr2 := NewTransaction(nil, nil, []byte(helpers.RandomString(helpers.RandomInt(0, 1024*1024))))
	tr3 := NewTransaction(nil, nil, []byte(helpers.RandomString(helpers.RandomInt(0, 1024*1024))))
	tr4 := NewTransaction(nil, nil, []byte(helpers.RandomString(helpers.RandomInt(0, 1024*1024))))

	b := new(Block)
	b.TransactionSlice = &TransactionSlice{*tr1, *tr2, *tr3, *tr4}

	mt := b.GenerateMerkelRoot()
	manual := helpers.SHA256(append(helpers.SHA256(append(tr1.Hash(), tr2.Hash()...)), helpers.SHA256(append(tr3.Hash(), tr4.Hash()...))...))

	if !reflect.DeepEqual(mt, manual) {
		t.Error("Merkel tree generation fails")
	}
}

func TestBlockMarshalling(t *testing.T) {
	kp := GenerateNewKeypair()
	tr := NewTransaction(kp.Public, nil, []byte(helpers.RandomString(helpers.RandomInt(0, 1024*1024))))

	tr.Header.Nonce = tr.GenerateNonce(helpers.ArrayOfBytes(TEST_TRANSACTION_POW_COMPLEXITY, TEST_POW_PREFIX))
	tr.Signature = tr.Sign(kp)

	// Create a block with the transaction
	block := NewBlock([]byte("previous block hash"))
	block.AddTransaction(tr)
	block.BlockHeader.Origin = kp.Public
	block.BlockHeader.MerkelRoot = block.GenerateMerkelRoot()
	block.BlockHeader.Nonce = 12345
	block.Signature = block.Sign(kp)

	// Test marshalling
	data, err := block.MarshalBinary()
	if err != nil {
		t.Error(err)
	}

	// Test unmarshalling
	newBlock := &Block{}
	err = newBlock.UnmarshalBinary(data)
	if err != nil {
		t.Error(err)
	}

	// Verify the unmarshalled block matches the original
	t.Logf("Original signature len: %d, sig: %v", len(block.Signature), block.Signature)
	t.Logf("Unmarshalled signature len: %d, sig: %v", len(newBlock.Signature), newBlock.Signature)
	t.Logf("Original tx slice len: %d", len(*block.TransactionSlice))
	t.Logf("Unmarshalled tx slice len: %d", len(*newBlock.TransactionSlice))
	t.Logf("Original tx: %v", block.TransactionSlice)
	t.Logf("Unmarshalled tx: %v", newBlock.TransactionSlice)

	// Debug origin field specifically
	t.Logf("Original origin len: %d, origin: %v", len(block.BlockHeader.Origin), block.BlockHeader.Origin)
	t.Logf("Unmarshalled origin len: %d, origin: %v", len(newBlock.BlockHeader.Origin), newBlock.BlockHeader.Origin)

	// Check individual fields since signature might be padded and fields might have leading zeros stripped
	originalHeader := *block.BlockHeader
	originalHeader.PrevBlock = helpers.StripByte(originalHeader.PrevBlock, 0)
	originalHeader.MerkelRoot = helpers.StripByte(originalHeader.MerkelRoot, 0)
	
	if !reflect.DeepEqual(*newBlock.BlockHeader, originalHeader) {
		t.Errorf("Block headers don't match: original=%v, unmarshalled=%v", originalHeader, newBlock.BlockHeader)
	}

	// Check transaction slice
	if len(*newBlock.TransactionSlice) != 1 {
		t.Errorf("Expected 1 transaction, got %d", len(*newBlock.TransactionSlice))
	} else {
		if !reflect.DeepEqual((*newBlock.TransactionSlice)[0], (*block.TransactionSlice)[0]) {
			t.Errorf("Transactions don't match")
		}
	}

	// Check signature - since it's padded to NETWORK_KEY_SIZE, we need to compare the relevant portion
	sig1 := helpers.FitBytesInto(block.Signature, NETWORK_KEY_SIZE)
	sig2 := helpers.FitBytesInto(newBlock.Signature, NETWORK_KEY_SIZE)
	if !reflect.DeepEqual(sig1, sig2) {
		t.Errorf("Signatures don't match")
	}
}

func TestBlockVerification(t *testing.T) {
	pow := helpers.ArrayOfBytes(TEST_BLOCK_POW_COMPLEXITY, TEST_POW_PREFIX)

	kp := GenerateNewKeypair()
	tr := NewTransaction(kp.Public, nil, []byte(helpers.RandomString(helpers.RandomInt(0, 1024))))

	tr.Header.Nonce = tr.GenerateNonce(helpers.ArrayOfBytes(TEST_TRANSACTION_POW_COMPLEXITY, TEST_POW_PREFIX))
	tr.Signature = tr.Sign(kp)

	block := NewBlock([]byte("previous block hash"))
	block.AddTransaction(tr)
	block.BlockHeader.Origin = kp.Public
	block.BlockHeader.MerkelRoot = block.GenerateMerkelRoot()
	block.BlockHeader.Nonce = block.GenerateNonce(pow) // Generate valid PoW nonce
	block.Signature = block.Sign(kp)

	if !block.VerifyBlock(pow) {
		t.Error("Block validation failing")
	}
}
