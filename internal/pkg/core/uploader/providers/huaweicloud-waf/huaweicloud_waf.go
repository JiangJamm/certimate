﻿package huaweicloudwaf

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	hcIam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	hcIamModel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	hcIamRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
	hcWaf "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1"
	hcWafModel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1/model"
	hcWafRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/waf/v1/region"
	xerrors "github.com/pkg/errors"

	"github.com/usual2970/certimate/internal/pkg/core/uploader"
	"github.com/usual2970/certimate/internal/pkg/utils/certs"
	hwsdk "github.com/usual2970/certimate/internal/pkg/vendors/huaweicloud-sdk"
)

type UploaderConfig struct {
	// 华为云 AccessKeyId。
	AccessKeyId string `json:"accessKeyId"`
	// 华为云 SecretAccessKey。
	SecretAccessKey string `json:"secretAccessKey"`
	// 华为云区域。
	Region string `json:"region"`
}

type UploaderProvider struct {
	config    *UploaderConfig
	sdkClient *hcWaf.WafClient
}

var _ uploader.Uploader = (*UploaderProvider)(nil)

func NewUploader(config *UploaderConfig) (*UploaderProvider, error) {
	if config == nil {
		panic("config is nil")
	}

	client, err := createSdkClient(
		config.AccessKeyId,
		config.SecretAccessKey,
		config.Region,
	)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to create sdk client")
	}

	return &UploaderProvider{
		config:    config,
		sdkClient: client,
	}, nil
}

func (u *UploaderProvider) Upload(ctx context.Context, certPem string, privkeyPem string) (res *uploader.UploadResult, err error) {
	// 解析证书内容
	certX509, err := certs.ParseCertificateFromPEM(certPem)
	if err != nil {
		return nil, err
	}

	// 遍历查询已有证书，避免重复上传
	// REF: https://support.huaweicloud.com/api-waf/ListCertificates.html
	// REF: https://support.huaweicloud.com/api-waf/ShowCertificate.html
	listCertificatesPage := int32(1)
	listCertificatesPageSize := int32(100)
	for {
		listCertificatesReq := &hcWafModel.ListCertificatesRequest{
			Page:     hwsdk.Int32Ptr(listCertificatesPage),
			Pagesize: hwsdk.Int32Ptr(listCertificatesPageSize),
		}
		listCertificatesResp, err := u.sdkClient.ListCertificates(listCertificatesReq)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to execute sdk request 'waf.ListCertificates'")
		}

		if listCertificatesResp.Items != nil {
			for _, certItem := range *listCertificatesResp.Items {
				showCertificateReq := &hcWafModel.ShowCertificateRequest{
					CertificateId: certItem.Id,
				}
				showCertificateResp, err := u.sdkClient.ShowCertificate(showCertificateReq)
				if err != nil {
					return nil, xerrors.Wrap(err, "failed to execute sdk request 'waf.ShowCertificate'")
				}

				var isSameCert bool
				if *showCertificateResp.Content == certPem {
					isSameCert = true
				} else {
					oldCertX509, err := certs.ParseCertificateFromPEM(*showCertificateResp.Content)
					if err != nil {
						continue
					}

					isSameCert = certs.EqualCertificate(certX509, oldCertX509)
				}

				// 如果已存在相同证书，直接返回已有的证书信息
				if isSameCert {
					return &uploader.UploadResult{
						CertId:   certItem.Id,
						CertName: certItem.Name,
					}, nil
				}
			}
		}

		if listCertificatesResp.Items == nil || len(*listCertificatesResp.Items) < int(listCertificatesPageSize) {
			break
		} else {
			listCertificatesPage++
		}
	}

	// 生成新证书名（需符合华为云命名规则）
	var certId, certName string
	certName = fmt.Sprintf("certimate-%d", time.Now().UnixMilli())

	// 创建证书
	// REF: https://support.huaweicloud.com/api-waf/CreateCertificate.html
	createCertificateReq := &hcWafModel.CreateCertificateRequest{
		Body: &hcWafModel.CreateCertificateRequestBody{
			Name:    certName,
			Content: certPem,
			Key:     privkeyPem,
		},
	}
	createCertificateResp, err := u.sdkClient.CreateCertificate(createCertificateReq)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to execute sdk request 'waf.CreateCertificate'")
	}

	certId = *createCertificateResp.Id
	certName = *createCertificateResp.Name
	return &uploader.UploadResult{
		CertId:   certId,
		CertName: certName,
	}, nil
}

func createSdkClient(accessKeyId, secretAccessKey, region string) (*hcWaf.WafClient, error) {
	projectId, err := getSdkProjectId(accessKeyId, secretAccessKey, region)
	if err != nil {
		return nil, err
	}

	auth, err := basic.NewCredentialsBuilder().
		WithAk(accessKeyId).
		WithSk(secretAccessKey).
		WithProjectId(projectId).
		SafeBuild()
	if err != nil {
		return nil, err
	}

	hcRegion, err := hcWafRegion.SafeValueOf(region)
	if err != nil {
		return nil, err
	}

	hcClient, err := hcWaf.WafClientBuilder().
		WithRegion(hcRegion).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, err
	}

	client := hcWaf.NewWafClient(hcClient)
	return client, nil
}

func getSdkProjectId(accessKeyId, secretAccessKey, region string) (string, error) {
	auth, err := global.NewCredentialsBuilder().
		WithAk(accessKeyId).
		WithSk(secretAccessKey).
		SafeBuild()
	if err != nil {
		return "", err
	}

	hcRegion, err := hcIamRegion.SafeValueOf(region)
	if err != nil {
		return "", err
	}

	hcClient, err := hcIam.IamClientBuilder().
		WithRegion(hcRegion).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return "", err
	}

	client := hcIam.NewIamClient(hcClient)

	request := &hcIamModel.KeystoneListProjectsRequest{
		Name: &region,
	}
	response, err := client.KeystoneListProjects(request)
	if err != nil {
		return "", err
	} else if response.Projects == nil || len(*response.Projects) == 0 {
		return "", errors.New("no project found")
	}

	return (*response.Projects)[0].Id, nil
}
