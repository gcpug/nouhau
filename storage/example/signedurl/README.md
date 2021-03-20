# SignedURL Example

tag:["google-cloud-storage", "signedurl"]

URLを知っていればアクセスできる期限付きのリソースアクセス権を提供でき、Cloud Storageに直接アップロード・ダウンロードできる [SignedURL](https://cloud.google.com/storage/docs/access-control/signed-urls) のExampleです。

ServiceAccountのSignBlobを利用して、SignedURLを発行する場合、以下のサンプルコードになる。
`projects/-/serviceAccounts/{SERVICE_ACCOUNT}` の部分が、少し不思議に見えるかもしれないが、 `-` の部分はそのままでOK

サンプルコードの全文は [github.com/sinmetalcraft](https://github.com/sinmetalcraft/gcpbox/blob/9365a8fefb20292b484d6608588ca700d60be288/storage/signedurl.go#L96) にあります。

``` Go
func (s *StorageSignedURLService) CreateSignedURL(ctx context.Context, bucket string, object string, method string, contentType string, headers []string, queryParameters url.Values, expires time.Time) (string, error) {
	opt := &storage.SignedURLOptions{
		GoogleAccessID:  s.ServiceAccountEmail,
		Method:          method,
		Expires:         expires,
		ContentType:     contentType,
		Headers:         headers,
		QueryParameters: queryParameters,
		Scheme:          storage.SigningSchemeV4,
		SignBytes: func(b []byte) ([]byte, error) {
			req := &credentialspb.SignBlobRequest{
				Name:    fmt.Sprintf("projects/-/serviceAccounts/%s", s.ServiceAccountEmail),
				Payload: b,
			}
			resp, err := s.IAMCredentialsClient.SignBlob(ctx, req)
			if err != nil {
				return nil, err
			}
			return resp.SignedBlob, nil
		},
	}
	u, err := storage.SignedURL(bucket, object, opt)
	if err != nil {
		return "", xerrors.Errorf("failed createSignedURL: sa=%s,bucket=%s,object=%s : %w", s.ServiceAccountEmail, bucket, object, err)
	}
	return u, nil
}
```

## Download Fileの名前とContent-Typeを指定する場合

SignedURLを利用してObjectをDownloadする場合、何も指定しなければ、ファイル名はCloud Storage上に保存しているObjectの最後の `/` の後ろの文字列になり、Content-Typeはmetadataに従います。
Objectの名前には全角文字列を使わずUUIDなどにしておいて、DBやObjectのmetaに本来のファイル名を保存しているケースが多いと思うので、その名前をDownload時のファイル名にする場合は、 `response-content-disposition` を利用します。
同じようにContent-Typeを指定したい場合、 `response-content-type` を利用します。
この仕様はDocumentにはっきり明記はされていませんが、 [この部分のノート](https://cloud.google.com/storage/docs/access-control/signed-urls-v2?hl=en#string-components) に `Note: Query String Parameters like response-content-disposition and response-content-type are not verified by the signature. To force a Content-Disposition or Content-Type in the response, set those parameters in the object metadata.` と書いてあることで、存在することが匂わされています。

``` Go
func (s *StorageSignedURLService) CreateDownloadURL(ctx context.Context, bucket string, object string, expires time.Time, param *CreateDownloadSignedURLParam) (string, error) {
	qp := url.Values{}
	if param != nil {
		var fileName string
		var cd string
		if len(param.DownloadFileName) > 0 {
			fileName = param.DownloadFileName
		}

		if param.Attachment {
			cd = fmt.Sprintf(`attachment;filename*=UTF-8''%s`, url.PathEscape(fileName))
			qp.Set("response-content-disposition", cd)
		}

		if len(param.DownloadContentType) > 0 {
			qp.Set("response-content-type", param.DownloadContentType)
		}
	}
	u, err := s.CreateSignedURL(ctx, bucket, object, http.MethodGet, "", []string{}, qp, expires)
	if err != nil {
		return "", xerrors.Errorf("failed CreateDownloadURL: %w", err)
	}
	return u, nil
}
```

サンプルコードの全文は [github.com/sinmetalcraft](https://github.com/sinmetalcraft/gcpbox/blob/9365a8fefb20292b484d6608588ca700d60be288/storage/signedurl.go#L70) にあります。
