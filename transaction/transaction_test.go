package transaction

import (
	"chain/common"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestTransactionSignAndRecover(t *testing.T) {
	// 1. 生成一个私钥用于测试
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("无法生成私钥: %v", err)
	}

	// 获取对应的以太坊地址
	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	ethAddr := crypto.PubkeyToAddress(*publicKey)
	expectedAddress := common.BytesToAddress(ethAddr.Bytes())

	fmt.Printf("生成的测试地址: %x\n", expectedAddress)

	// 2. 创建测试交易
	tx := &Transaction{
		Txdata: Txdata{
			To:       common.Address{0x1},
			GasPrice: 20000000000, // 20 Gwei
			Nonce:    1,
			ChainID:  big.NewInt(1),
			GasLimit: 21000,
			Value:    big.NewInt(1000000000000000000), // 1 ETH
			Data:     []byte{},
		},
		Signature: nil, // 初始时没有签名
	}

	// 3. 签名交易
	sig := tx.SetSignature(privateKey)
	if len(sig) == 0 {
		t.Fatal("交易签名失败")
	}

	if tx.Signature != nil && len(tx.Signature) == 65 {
		v, r, s := tx.extractSignature()
		fmt.Printf("签名成功:\n")
		fmt.Printf("V: %v\n", v)
		fmt.Printf("R: %x\n", r.Bytes())
		fmt.Printf("S: %x\n", s.Bytes())
	}

	// 4. 从交易中恢复发送者地址
	recoveredAddress := tx.From()
	if recoveredAddress == (common.Address{}) {
		t.Fatal("无法恢复发送者地址")
	}

	fmt.Printf("恢复的地址: %x\n", recoveredAddress)

	// 5. 验证恢复的地址是否正确
	if recoveredAddress != expectedAddress {
		t.Errorf("地址恢复错误, 期望: %x, 得到: %x", expectedAddress, recoveredAddress)
	} else {
		fmt.Println("地址恢复成功, 验证通过!")
	}
}

// 附加测试：测试修改交易后，地址恢复是否失败
func TestTransactionTampering(t *testing.T) {
	// 1. 生成一个私钥用于测试
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("无法生成私钥: %v", err)
	}

	// 2. 创建一个测试交易
	tx := &Transaction{
		Txdata: Txdata{
			To:       common.Address{0x1},
			GasPrice: 20000000000,
			Nonce:    1,
			ChainID:  big.NewInt(1),
			GasLimit: 21000,
			Value:    big.NewInt(1000000000000000000), // 1 ETH
			Data:     []byte{},
		},
		Signature: nil,
	}

	// 3. 签名交易
	tx.SetSignature(privateKey)

	// 保存原始恢复的地址
	originalAddress := tx.From()

	// 4. 修改交易金额
	tx.Value = big.NewInt(2000000000000000000) // 2 ETH

	// 5. 尝试恢复地址
	tamperedAddress := tx.From()

	// 6. 验证恢复的地址是否不同
	if tamperedAddress == originalAddress {
		t.Error("安全漏洞: 修改交易后地址仍然能够恢复!")
	} else {
		fmt.Println("安全测试通过: 修改交易后地址恢复失败，证明交易完整性!")
	}
}

// 🌟 修复：序列化和哈希测试
func TestTransactionSerializeDeserialize(t *testing.T) {
	// 1. 创建一个测试交易
	tx := &Transaction{
		Txdata: Txdata{
			To:       common.Address{0x2},
			GasPrice: 30000000000,
			Nonce:    5,
			ChainID:  big.NewInt(1),
			GasLimit: 21000,
			Value:    big.NewInt(500000000000000000), // 0.5 ETH
			Data:     []byte("test data"),
		},
		Signature: nil,
	}

	// 🌟 修复：不需要手动设置V,R,S，因为结构体中没有这些字段

	// 2. 序列化交易
	data, err := tx.Serialize()
	if err != nil {
		t.Fatalf("序列化交易失败: %v", err)
	}

	if len(data) == 0 {
		t.Fatal("序列化后数据为空")
	}

	fmt.Printf("序列化交易成功，长度: %d 字节\n", len(data))

	// 3. 验证GetHash函数
	hash := tx.GetHash()
	if hash == (common.Hash{}) {
		t.Fatal("计算交易哈希失败")
	}

	fmt.Printf("交易哈希: %x\n", hash)

	// 4. 测试多次调用GetHash是否一致
	hash2 := tx.GetHash()
	if hash != hash2 {
		t.Error("多次调用GetHash结果不一致")
	} else {
		fmt.Println("多次调用GetHash结果一致")
	}
}

// 测试多交易盒子排序
func TestTransactionBoxSorting(t *testing.T) {
	// 1. 创建多个交易盒子
	box1 := Txbox{
		GasPrice: 10000000000,     // 10 Gwei
		Tx:       []Transaction{}, // 空交易列表
	}

	box2 := Txbox{
		GasPrice: 20000000000, // 20 Gwei
		Tx:       []Transaction{},
	}

	box3 := Txbox{
		GasPrice: 15000000000, // 15 Gwei
		Tx:       []Transaction{},
	}

	// 2. 创建盒子列表
	boxes := Boxes{box1, box2, box3}

	// 3. 测试比较函数
	// GasPrice高的应该排在前面（返回true）
	if !boxes.Less(1, 0) {
		t.Error("排序函数错误: box2(20 Gwei)应该比box1(10 Gwei)有更高的GasPrice")
	}

	if !boxes.Less(1, 2) {
		t.Error("排序函数错误: box2(20 Gwei)应该比box3(15 Gwei)有更高的GasPrice")
	}

	if !boxes.Less(2, 0) {
		t.Error("排序函数错误: box3(15 Gwei)应该比box1(10 Gwei)有更高的GasPrice")
	}

	// 4. 打印长度
	fmt.Printf("交易盒子数量: %d\n", boxes.Len())

	fmt.Println("✅ 交易盒子排序测试通过!")
}

// 🌟 修复：NewTransaction构造函数测试
func TestNewTransaction(t *testing.T) {
	// 🌟 修复：直接调用NewTransaction，不需要创建空的tx
	newTx := (&Transaction{}).NewTransaction(
		common.Address{0x1}, // to
		20000000000,         // gasPrice
		1,                   // nonce
	)

	// 验证字段设置
	assert := func(condition bool, message string) {
		if !condition {
			t.Error(message)
		}
	}

	assert(newTx.To == common.Address{0x1}, "To地址应该正确设置")
	assert(newTx.GasPrice == 20000000000, "GasPrice应该正确设置")
	assert(newTx.Nonce == 1, "Nonce应该正确设置")
	assert(newTx.Signature == nil, "初始签名应该为nil")
	assert(newTx.ChainID.Cmp(big.NewInt(1)) == 0, "ChainID应该正确设置")
	assert(newTx.GasLimit == 21000, "GasLimit应该有默认值")
	assert(newTx.Value.Cmp(big.NewInt(0)) == 0, "Value应该有默认值0")

	fmt.Println("✅ NewTransaction测试通过!")
}

// 🌟 新增：测试签名和验证的完整流程
func TestTransactionSignAndVerifyFlow(t *testing.T) {
	// 1. 创建私钥
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("无法生成私钥: %v", err)
	}

	// 2. 创建交易
	tx := &Transaction{
		Txdata: Txdata{
			To:       common.Address{0x1, 0x2, 0x3},
			GasPrice: 25000000000, // 25 Gwei
			Nonce:    42,
			ChainID:  big.NewInt(1),
			GasLimit: 50000,
			Value:    big.NewInt(1500000000000000000), // 1.5 ETH
			Data:     []byte("Hello, Blockchain!"),
		},
		Signature: nil,
	}

	// 3. 签名前验证地址为空
	addrBeforeSign := tx.From()
	if addrBeforeSign != (common.Address{}) {
		t.Error("签名前应该无法恢复地址")
	}

	// 4. 签名交易
	signature := tx.SetSignature(privateKey)
	if len(signature) != 65 {
		t.Fatalf("签名长度应该是65字节，实际: %d", len(signature))
	}

	// 5. 验证签名后能正确恢复地址
	recoveredAddr := tx.From()
	if recoveredAddr == (common.Address{}) {
		t.Fatal("签名后应该能恢复地址")
	}

	// 6. 验证恢复的地址与私钥匹配
	expectedAddr := crypto.PubkeyToAddress(privateKey.PublicKey)
	if recoveredAddr != common.BytesToAddress(expectedAddr.Bytes()) {
		t.Errorf("恢复的地址与预期不符")
	}

	// 7. 验证哈希计算
	hash := tx.GetHash()
	if hash == (common.Hash{}) {
		t.Error("哈希计算失败")
	}

	fmt.Printf("✅ 完整流程测试通过!\n")
	fmt.Printf("  - 交易哈希: %x\n", hash)
	fmt.Printf("  - 发送者地址: %x\n", recoveredAddr)
	fmt.Printf("  - 签名长度: %d bytes\n", len(signature))
}

// 🌟 新增：性能基准测试
func BenchmarkGetHash(b *testing.B) {
	tx := &Transaction{
		Txdata: Txdata{
			To:       common.Address{0x1},
			GasPrice: 20000000000,
			Nonce:    1,
			ChainID:  big.NewInt(1),
			GasLimit: 21000,
			Value:    big.NewInt(1000),
			Data:     []byte{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tx.GetHash()
	}
}

func BenchmarkSignAndRecover(b *testing.B) {
	privateKey, _ := crypto.GenerateKey()

	tx := &Transaction{
		Txdata: Txdata{
			To:       common.Address{0x1},
			GasPrice: 20000000000,
			Nonce:    1,
			ChainID:  big.NewInt(1),
			GasLimit: 21000,
			Value:    big.NewInt(1000),
			Data:     []byte{},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tx.SetSignature(privateKey)
		_ = tx.From()
	}
}
