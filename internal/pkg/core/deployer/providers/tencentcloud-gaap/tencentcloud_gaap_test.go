package tencentcloudgaap_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	provider "github.com/usual2970/certimate/internal/pkg/core/deployer/providers/tencentcloud-gaap"
)

var (
	fInputCertPath string
	fInputKeyPath  string
	fSecretId      string
	fSecretKey     string
	fProxyGroupId  string
	fProxyId       string
	fListenerId    string
)

func init() {
	argsPrefix := "CERTIMATE_DEPLOYER_TENCENTCLOUDCDN_"

	flag.StringVar(&fInputCertPath, argsPrefix+"INPUTCERTPATH", "", "")
	flag.StringVar(&fInputKeyPath, argsPrefix+"INPUTKEYPATH", "", "")
	flag.StringVar(&fSecretId, argsPrefix+"SECRETID", "", "")
	flag.StringVar(&fSecretKey, argsPrefix+"SECRETKEY", "", "")
	flag.StringVar(&fProxyGroupId, argsPrefix+"PROXYGROUPID", "", "")
	flag.StringVar(&fProxyId, argsPrefix+"PROXYID", "", "")
	flag.StringVar(&fListenerId, argsPrefix+"LISTENERID", "", "")
}

/*
Shell command to run this test:

	go test -v ./tencentcloud_gaap_test.go -args \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_INPUTCERTPATH="/path/to/your-input-cert.pem" \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_INPUTKEYPATH="/path/to/your-input-key.pem" \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_SECRETID="your-secret-id" \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_SECRETKEY="your-secret-key" \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_PROXYGROUPID="your-gaap-group-id" \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_PROXYID="your-gaap-group-id" \
	--CERTIMATE_DEPLOYER_TENCENTCLOUDGAAP_LISTENERID="your-clb-listener-id"
*/
func TestDeploy(t *testing.T) {
	flag.Parse()

	t.Run("Deploy_ToListener", func(t *testing.T) {
		t.Log(strings.Join([]string{
			"args:",
			fmt.Sprintf("INPUTCERTPATH: %v", fInputCertPath),
			fmt.Sprintf("INPUTKEYPATH: %v", fInputKeyPath),
			fmt.Sprintf("SECRETID: %v", fSecretId),
			fmt.Sprintf("SECRETKEY: %v", fSecretKey),
			fmt.Sprintf("PROXYGROUPID: %v", fProxyGroupId),
			fmt.Sprintf("PROXYID: %v", fProxyId),
			fmt.Sprintf("LISTENERID: %v", fListenerId),
		}, "\n"))

		deployer, err := provider.NewDeployer(&provider.DeployerConfig{
			SecretId:     fSecretId,
			SecretKey:    fSecretKey,
			ResourceType: provider.RESOURCE_TYPE_LISTENER,
			ProxyGroupId: fProxyGroupId,
			ProxyId:      fProxyId,
			ListenerId:   fListenerId,
		})
		if err != nil {
			t.Errorf("err: %+v", err)
			return
		}

		fInputCertData, _ := os.ReadFile(fInputCertPath)
		fInputKeyData, _ := os.ReadFile(fInputKeyPath)
		res, err := deployer.Deploy(context.Background(), string(fInputCertData), string(fInputKeyData))
		if err != nil {
			t.Errorf("err: %+v", err)
			return
		}

		t.Logf("ok: %v", res)
	})
}
