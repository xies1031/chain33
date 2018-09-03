package contract

import (
	"sync"

	"github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/cmd/autotest/testcase"
)

type TestTokenConfig struct {
	TokenPreCreateCaseArr    []testcase.TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []testcase.TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []testcase.TransferCase          `toml:"TransferCase,omitempty"`
	WithdrawCaseArr          []testcase.WithdrawCase          `toml:"WithdrawCase,omitempty"`
}

func (caseConf *TestTokenConfig) RunTest(caseFile string, wg *sync.WaitGroup) {

	defer wg.Done()

	fLog := fileLog.New("module", "Token")
	tLog := stdLog.New("module", "Token")
	if _, err := toml.DecodeFile(caseFile, &caseConf); err != nil {

		tLog.Error("ErrTomlDecode", "Error", err.Error())
		return
	}

	tester := testcase.NewTestOperator(fLog, tLog)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()

	for i := range caseConf.TokenPreCreateCaseArr {

		tester.AddCase(&caseConf.TokenPreCreateCaseArr[i])
	}

	for i := range caseConf.TokenFinishCreateCaseArr {

		tester.AddCase(&caseConf.TokenFinishCreateCaseArr[i])
	}

	for i := range caseConf.TransferCaseArr {

		tester.AddCase(&caseConf.TransferCaseArr[i])
	}

	for i := range caseConf.WithdrawCaseArr {

		tester.AddCase(&caseConf.WithdrawCaseArr[i])
	}

	tester.WaitTest()

}