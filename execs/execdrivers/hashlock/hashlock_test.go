package coins

import (
	"context"
	crand "crypto/rand"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
	"google.golang.org/grpc"
)

var conn *grpc.ClientConn
var r *rand.Rand
var c types.GrpcserviceClient
var ErrTest = errors.New("ErrTest")

var secret []byte
var wrongsecret []byte
var anothersec []byte //used in send case

var addrexec *account.Address

var locktime = minLockTime + 10 // bigger than minLockTime defined in hashlock.go

const (
	accountindexA = 0
	accountindexB = 1
	accountMax    = 2
)

const (
	defaultAmount = 1e10
	fee           = 1e6
	lockAmount    = 1e8
)

const (
	onlyshow     = 0
	onlycheck    = 1
	showandcheck = 2
)

const secretLen = 32

var addr [accountMax]string
var privkey [accountMax]crypto.PrivKey

var currBalanceA int64
var currBalanceB int64

func init() {
	var err error
	conn, err = grpc.Dial("localhost:8802", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
	c = types.NewGrpcserviceClient(conn)
	secret = make([]byte, secretLen)
	wrongsecret = make([]byte, secretLen)
	anothersec = make([]byte, secretLen)
	crand.Read(secret)
	crand.Read(wrongsecret)
	crand.Read(anothersec)
	addrexec = account.ExecAddress("hashlock")
}

func TestInitAccount(t *testing.T) {
	fmt.Println("TestInitAccount start")
	defer fmt.Println("TestInitAccount end\n")

	var label [accountMax]string
	var params types.ReqWalletImportPrivKey

	privGenesis := getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	for index := 0; index < accountMax; index++ {
		addr[index], privkey[index] = genaddress()
		//fmt.Println("privkey: ", common.ToHex(privkey[index].Bytes()))
		label[index] = strconv.Itoa(int(time.Now().UnixNano()))
		params = types.ReqWalletImportPrivKey{Privkey: common.ToHex(privkey[index].Bytes()), Label: label[index]}
		_, err := c.ImportPrivKey(context.Background(), &params)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, 0) {
			t.Error(ErrTest)
			return
		}
		time.Sleep(5 * time.Second)
	}

	for index := 0; index < accountMax; index++ {
		err := sendtoaddress(c, privGenesis, addr[index], defaultAmount)
		if err != nil {
			fmt.Println(err)
			time.Sleep(time.Second)
			t.Error(err)
			return
		}
		time.Sleep(5 * time.Second)
		if !showOrCheckAcc(c, addr[index], showandcheck, defaultAmount) {
			t.Error(ErrTest)
			return
		}
	}
	currBalanceA = defaultAmount
	currBalanceB = defaultAmount
}

func TestHashlock(t *testing.T) {

	fmt.Println("TestHashlock start")
	defer fmt.Println("TestHashlock end\n")

	//1. step1 发送余额给合约
	err := sendtoaddress(c, privkey[accountindexA], addrexec.String(), lockAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	err = lock(secret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)

	currBalanceA -= lockAmount + 2*fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
}

func TestHashunlock(t *testing.T) {
	fmt.Println("TestHashunlock start")
	defer fmt.Println("TestHashunlock end\n")
	//not sucess as time not enough
	time.Sleep(5 * time.Second)
	err := unlock(secret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	//尝试取钱
	err = sendtoaddress(c, privkey[accountindexA], addrexec.String(), 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= 2 * fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	//not success as secret is not right
	time.Sleep(70 * time.Second)
	err = unlock(wrongsecret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexA], addrexec.String(), 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= 2 * fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}

	//success
	time.Sleep(5 * time.Second)
	err = unlock(secret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexA], addrexec.String(), 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA = currBalanceA - 2*fee + lockAmount
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
}

func TestHashsend(t *testing.T) {
	fmt.Println("TstHashsend start")
	defer fmt.Println("TstHashsend end\n")
	//lock it again &send failed as secret is not right
	//send failed as secret is not right
	err := sendtoaddress(c, privkey[accountindexA], addrexec.String(), lockAmount)
	if err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	err = lock(anothersec)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= lockAmount + 2*fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	time.Sleep(5 * time.Second)
	err = send(wrongsecret)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexB], addrexec.String(), 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= fee
	currBalanceB -= fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexB], showandcheck, currBalanceB) {
		t.Error(ErrTest)
		return
	}
	//success
	time.Sleep(5 * time.Second)
	err = send(anothersec)
	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(5 * time.Second)
	err = sendtoaddress(c, privkey[accountindexB], addrexec.String(), 0-lockAmount)
	if err != nil {
		fmt.Println("err")
	}
	time.Sleep(5 * time.Second)
	currBalanceA -= fee
	currBalanceB = currBalanceB + lockAmount - fee
	if !showOrCheckAcc(c, addr[accountindexA], showandcheck, currBalanceA) {
		t.Error(ErrTest)
		return
	}
	if !showOrCheckAcc(c, addr[accountindexB], showandcheck, currBalanceB) {
		t.Error(ErrTest)
		return
	}
	//lock it again & failed as overtime

}

func lock(secret []byte) error {
	vlock := &types.HashlockAction_Hlock{&types.HashlockLock{Amount: lockAmount, Time: int64(locktime), Hash: common.Sha256(secret), ToAddress: addr[accountindexB], ReturnAddress: addr[accountindexA]}}
	//fmt.Println(vlock)
	transfer := &types.HashlockAction{Value: vlock, Ty: types.HashlockActionLock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: addr[accountindexB]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[accountindexA])
	// Contact the server and print out its response.
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}
func unlock(secret []byte) error {

	vunlock := &types.HashlockAction_Hunlock{&types.HashlockUnlock{Secret: secret}}
	transfer := &types.HashlockAction{Value: vunlock, Ty: types.HashlockActionUnlock}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: addr[accountindexB]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[accountindexA])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func send(secret []byte) error {

	vsend := &types.HashlockAction_Hsend{&types.HashlockSend{Secret: secret}}
	transfer := &types.HashlockAction{Value: vsend, Ty: types.HashlockActionSend}
	tx := &types.Transaction{Execer: []byte("hashlock"), Payload: types.Encode(transfer), Fee: fee, To: addr[accountindexB]}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, privkey[accountindexA])
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func showOrCheckAcc(c types.GrpcserviceClient, addr string, sorc int, balance int64) bool {
	req := &types.ReqNil{}
	accs, err := c.GetAccounts(context.Background(), req)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(accs.Wallets); i++ {
		wallet := accs.Wallets[i]
		if wallet.Acc.Addr == addr {
			if sorc != onlycheck {
				fmt.Println(wallet)
			}
			if sorc != onlyshow {
				if balance != wallet.Acc.Balance {
					fmt.Println(balance, wallet.Acc.Balance)
					return false
				}
			}
			return true
		}
	}
	if sorc != onlyshow {
		return false
	} else {
		return true
	}
}

func showAccount(c types.GrpcserviceClient, addr string) {
	req := &types.ReqNil{}
	accs, err := c.GetAccounts(context.Background(), req)
	if err != nil {
		panic(err)
	}
	for i := 0; i < len(accs.Wallets); i++ {
		wallet := accs.Wallets[i]
		if wallet.Acc.Addr == addr {
			fmt.Println(wallet)
			break
		}
	}
}

func checkAccount(balance int64, frozen int64, wallet *types.WalletAccount) bool {
	return ((balance == wallet.Acc.Balance) && (frozen == wallet.Acc.Frozen))
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func sendtoaddress(c types.GrpcserviceClient, priv crypto.PrivKey, to string, amount int64) error {
	//defer conn.Close()
	//fmt.Println("sign key privkey: ", common.ToHex(priv.Bytes()))
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: fee, To: to}
	tx.Nonce = r.Int63()
	tx.Sign(types.SECP256K1, priv)
	// Contact the server and print out its response.
	reply, err := c.SendTransaction(context.Background(), tx)
	if err != nil {
		fmt.Println("err", err)
		return err
	}
	if !reply.IsOk {
		fmt.Println("err = ", reply.GetMsg())
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func getAccounts() (*types.WalletAccounts, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetAccounts(context.Background(), v)
}

func getlastheader() (*types.Header, error) {
	c := types.NewGrpcserviceClient(conn)
	v := &types.ReqNil{}
	return c.GetLastHeader(context.Background(), v)
}
