﻿package huaweicloudelb

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
	"github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/global"
	hcElb "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3"
	hcElbModel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3/model"
	hcElbRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/elb/v3/region"
	hcIam "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3"
	hcIamModel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/model"
	hcIamRegion "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/iam/v3/region"
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
	sdkClient *hcElb.ElbClient
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
	// REF: https://support.huaweicloud.com/api-elb/ListCertificates.html
	listCertificatesLimit := int32(2000)
	var listCertificatesMarker *string = nil
	for {
		listCertificatesReq := &hcElbModel.ListCertificatesRequest{
			Limit:  hwsdk.Int32Ptr(listCertificatesLimit),
			Marker: listCertificatesMarker,
			Type:   &[]string{"server"},
		}
		listCertificatesResp, err := u.sdkClient.ListCertificates(listCertificatesReq)
		if err != nil {
			return nil, xerrors.Wrap(err, "failed to execute sdk request 'elb.ListCertificates'")
		}

		if listCertificatesResp.Certificates != nil {
			for _, certDetail := range *listCertificatesResp.Certificates {
				var isSameCert bool
				if certDetail.Certificate == certPem {
					isSameCert = true
				} else {
					oldCertX509, err := certs.ParseCertificateFromPEM(certDetail.Certificate)
					if err != nil {
						continue
					}

					isSameCert = certs.EqualCertificate(certX509, oldCertX509)
				}

				// 如果已存在相同证书，直接返回已有的证书信息
				if isSameCert {
					return &uploader.UploadResult{
						CertId:   certDetail.Id,
						CertName: certDetail.Name,
					}, nil
				}
			}
		}

		if listCertificatesResp.Certificates == nil || len(*listCertificatesResp.Certificates) < int(listCertificatesLimit) {
			break
		} else {
			listCertificatesMarker = listCertificatesResp.PageInfo.NextMarker
		}
	}

	// 获取项目 ID
	// REF: https://support.huaweicloud.com/api-iam/iam_06_0001.html
	projectId, err := getSdkProjectId(u.config.AccessKeyId, u.config.SecretAccessKey, u.config.Region)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to get SDK project id")
	}

	// 生成新证书名（需符合华为云命名规则）
	var certId, certName string
	certName = fmt.Sprintf("certimate-%d", time.Now().UnixMilli())

	// 创建新证书
	// REF: https://support.huaweicloud.com/api-elb/CreateCertificate.html
	createCertificateReq := &hcElbModel.CreateCertificateRequest{
		Body: &hcElbModel.CreateCertificateRequestBody{
			Certificate: &hcElbModel.CreateCertificateOption{
				ProjectId:   hwsdk.StringPtr(projectId),
				Name:        hwsdk.StringPtr(certName),
				Certificate: hwsdk.StringPtr(certPem),
				PrivateKey:  hwsdk.StringPtr(privkeyPem),
			},
		},
	}
	createCertificateResp, err := u.sdkClient.CreateCertificate(createCertificateReq)
	if err != nil {
		return nil, xerrors.Wrap(err, "failed to execute sdk request 'elb.CreateCertificate'")
	}

	certId = createCertificateResp.Certificate.Id
	certName = createCertificateResp.Certificate.Name
	return &uploader.UploadResult{
		CertId:   certId,
		CertName: certName,
	}, nil
}

func createSdkClient(accessKeyId, secretAccessKey, region string) (*hcElb.ElbClient, error) {
	if region == "" {
		region = "cn-north-4" // ELB 服务默认区域：华北四北京
	}

	auth, err := basic.NewCredentialsBuilder().
		WithAk(accessKeyId).
		WithSk(secretAccessKey).
		SafeBuild()
	if err != nil {
		return nil, err
	}

	hcRegion, err := hcElbRegion.SafeValueOf(region)
	if err != nil {
		return nil, err
	}

	hcClient, err := hcElb.ElbClientBuilder().
		WithRegion(hcRegion).
		WithCredential(auth).
		SafeBuild()
	if err != nil {
		return nil, err
	}

	client := hcElb.NewElbClient(hcClient)
	return client, nil
}

func getSdkProjectId(accessKeyId, secretAccessKey, region string) (string, error) {
	if region == "" {
		region = "cn-north-4" // IAM 服务默认区域：华北四北京
	}

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
