# Customer Supplied Encryption Keys VS Customer Managed Encryption Keys

[Customer Supplied Encryption Keys](https://cloud.google.com/storage/docs/encryption/customer-supplied-keys?hl=en)と[Customer Managed Encryption Keys](https://cloud.google.com/storage/docs/encryption/customer-managed-keys?hl=en) の比較です。
Customer Managed Encryption Keysの方が後継の機能になるため、基本的にはCustomer Managed Encryption Keysの方が便利です。

## Customer Supplied Encryption Keys

Cloud Storageへのアップロード/ダウンロード時に任意に作成したAES256の鍵を利用する方式です。
自分で用意した任意の鍵を使うことができますが、鍵の管理は自分で行う必要があります。

### Sample Code

独自の鍵はDBに保存するなど、様々な方法が考えられますが、 https://github.com/googlecodelabs/cloud-kms-java-codelab/tree/master/src/main/java/com/example/getstarted に鍵自体をCloud KMSで暗号化し、ObjectのMetadataに置くというサンプルがあるので、これを参考にGoで書いたのが以下です。
全容は https://github.com/sinmetal/gcs_sample

``` Go
// Encrypt is 指定したCloud KMSの鍵で暗号化する
// keyName format: "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s
func (s *CSEKService) Encrypt(ctx context.Context, keyName string, plaintext string) (ciphertext string, cryptoKey string, err error) {
	ctx = trace.StartSpan(ctx, "encryption/csek/encrypt")
	defer trace.EndSpan(ctx, err)

	response, err := s.kms.Projects.Locations.KeyRings.CryptoKeys.Encrypt(keyName, &cloudkms.EncryptRequest{
		Plaintext: plaintext,
	}).Do()
	if err != nil {
		return "", "", fmt.Errorf("encrypt: failed to encrypt. CryptoKey=%s : %w", keyName, err)
	}

	return response.Ciphertext, response.Name, nil
}

// Encrypt is 指定したCloud KMSの鍵で復号化する
// keyName format: "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s
func (s *CSEKService) Decrypt(ctx context.Context, keyName string, ciphertext string) (plaintext string, err error) {
	ctx = trace.StartSpan(ctx, "encryption/csek/decrypt")
	defer trace.EndSpan(ctx, err)

	response, err := s.kms.Projects.Locations.KeyRings.CryptoKeys.Decrypt(keyName, &cloudkms.DecryptRequest{
		Ciphertext: ciphertext,
	}).Do()
	if err != nil {
		return "", fmt.Errorf("decrypt: failed to decrypt. CryptoKey=%s : %w", keyName, err)
	}

	return response.Plaintext, nil
}

// Upload is Cloud Storageに指定されたファイルをアップロードする
// アップロードする時にcustomer-supplied encryption keyとしてencryptionKeyを利用する
// encryptionKeyはkeyNameで指定されたCloud KMS Keyを利用して暗号化し、Object.Metadata[wDEK]として保存する
//
// keyName format: "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s
// encryptionKey: 256 bit (32 byte) AES encryption key
func (s *CSEKService) Upload(ctx context.Context, keyName string, bucketName string, objectName string, encryptionKey []byte, file []byte) (size int, err error) {
	ctx = trace.StartSpan(ctx, "encryption/csek/upload")
	defer trace.EndSpan(ctx, err)

	obj := s.gcs.Bucket(bucketName).Object(objectName).Key(encryptionKey)
	w := obj.NewWriter(ctx)

	ekt := base64.StdEncoding.EncodeToString(encryptionKey)
	chiphertext, cryptKey, err := s.Encrypt(ctx, keyName, ekt)
	if err != nil {
		return 0, fmt.Errorf("failed encrypt: %w", err)
	}

	metadata := map[string]string{}
	metadata["wDEK"] = chiphertext
	metadata["cryptKey"] = cryptKey // keyVersionを保持するために入れる
	w.Metadata = metadata
	size, err = w.Write(file)
	if err != nil {
		return 0, fmt.Errorf("failed gcs.write: %w", err)
	}

	if err := w.Close(); err != nil {
		return size, fmt.Errorf("file writer close error: %w", err)
	}

	return size, nil
}

// Download is Cloud Storageから指定されたファイルをダウンロードする
// ダウンロードする時にcustomer-supplied encryption keyとして、Object.Metadata[wDEK]から取得した値をkeyNameで指定されたCloud KMS Keyで復号化して利用する
//
// keyName format: "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s
func (s *CSEKService) Download(ctx context.Context, keyName string, bucketName string, objectName string) (data []byte, attrs *storage.ObjectAttrs, err error) {
	ctx = trace.StartSpan(ctx, "encryption/csek/download")
	defer trace.EndSpan(ctx, err)

	rc, attrs, err := s.NewDownloader(ctx, keyName, bucketName, objectName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed object.NewReader: %w", err)
	}
	defer func() {
		if err := rc.Close(); err != nil {
			// noop
		}
	}()

	data, err = ioutil.ReadAll(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed object.Read: %w", err)
	}
	return data, attrs, nil
}

func (s *CSEKService) NewDownloader(ctx context.Context, keyName string, bucketName string, objectName string) (w io.ReadCloser, attrs *storage.ObjectAttrs, err error) {
	ctx = trace.StartSpan(ctx, "encryption/csek/newDownloader")
	defer trace.EndSpan(ctx, err)

	obj := s.gcs.Bucket(bucketName).Object(objectName)
	attrs, err = obj.Attrs(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed read object.Attrs: %w", err)
	}
	encryptedSecretKey := attrs.Metadata["wDEK"]
	if len(encryptedSecretKey) < 1 {
		return nil, nil, fmt.Errorf("not found encryptedSecretKey from object.Metadata[wDEK]")
	}

	plainttext, err := s.Decrypt(ctx, keyName, encryptedSecretKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed decrpyt encryptedSecretKey: %w", err)
	}
	secretKey, err := base64.StdEncoding.DecodeString(plainttext)
	if err != nil {
		return nil, nil, fmt.Errorf("failed base64.Decode encryptedSecretKey: %w", err)
	}

	rc, err := obj.Key(secretKey).NewReader(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed object.NewReader: %w", err)
	}

	return rc, attrs, nil
}
```

#### ObjectのCopy

Customer Supplied Encryption Keyを利用するとCloud Storage Transfer ServiceなどObjectのCopyを行う機能が使えなくなります。
そのため、ObjectのCopyを行いたい場合、自力で行う必要があります。
[Objects: copy](https://cloud.google.com/storage/docs/json_api/v1/objects/copy) を利用すれば、Bucketを越えたObjectのCopyをCloud Storage側で行うことができます。
以下のサンプルコードは別BucketへのCopyを行うサンプルコードです。

``` Go
// Copy is src側,dst側それぞれにCSEKを渡して、向こうでCopyしてもらう
func (s *CSEKService) Copy(ctx context.Context, dstBucket string, srcBucket string, objectName string, keyName string) (err error) {
	ctx = trace.StartSpan(ctx, "encryption/csek/copy")
	defer trace.EndSpan(ctx, err)

	obj := s.gcs.Bucket(srcBucket).Object(objectName)
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		return fmt.Errorf("failed read object.Attrs: %w", err)
	}
	encryptedSecretKey := attrs.Metadata["wDEK"]
	if len(encryptedSecretKey) < 1 {
		return fmt.Errorf("not found encryptedSecretKey from object.Metadata[wDEK]")
	}

	plainttext, err := s.Decrypt(ctx, keyName, encryptedSecretKey)
	if err != nil {
		return fmt.Errorf("failed decrpyt encryptedSecretKey: %w", err)
	}
	secretKey, err := base64.StdEncoding.DecodeString(plainttext)
	if err != nil {
		return fmt.Errorf("failed base64.Decode encryptedSecretKey: %w", err)
	}

	src := obj.Key(secretKey)
	copier := s.gcs.Bucket(dstBucket).Object(objectName).Key(secretKey).CopierFrom(src)
	metadata := map[string]string{}
	metadata["wDEK"] = encryptedSecretKey
	metadata["cryptKey"] = keyName // keyVersionを保持するために入れる
	copier.Metadata = metadata
	_, err = copier.Run(ctx)
	if err != nil {
		return err
	}
	return nil
}

```

## Customer Managed Encryption Keys

Customer Managed Encryption KeysはCloud KMSのKeyを作成し、それを指定するだけでCloud Storage側で暗号化に使ってくれる機能です。
使うのは簡単ですが、Cloud Storage Service Agentに対して権限を付与するなどの[準備](https://cloud.google.com/storage/docs/encryption/using-customer-managed-keys#service-agent-access)が必要です。

### Sample Code

Customer Managed Encryption Keysの場合、[Bucketに対してDefaultで利用するCloud KMSのKeyを指定できる](https://cloud.google.com/storage/docs/encryption/using-customer-managed-keys#add-default-key)ので、その場合、アップロード/ダウンロードは特別なコードは必要ありません。
Cloud KMSの権限が追加で必要になるだけです。

``` Go
// Upload is Cloud Storageにfileをアップロードする
// CMEKとしてBucket Default Keyを指定しているので、コード上はただアップロードしてるだけ
func (s *CMEKService) Upload(ctx context.Context, bucketName string, objectName string, file []byte) (size int, err error) {
	ctx = trace.StartSpan(ctx, "encryption/cmek/upload")
	defer trace.EndSpan(ctx, err)

	// bucket default keyを指定してるので、普通にUploadしている
	// https://cloud.google.com/storage/docs/encryption/using-customer-managed-keys?hl=en#add-default-key
	obj := s.gcs.Bucket(bucketName).Object(objectName)
	w := obj.NewWriter(ctx)

	size, err = w.Write(file)
	if err != nil {
		return 0, fmt.Errorf("failed gcs.write: %w", err)
	}

	if err := w.Close(); err != nil {
		return size, fmt.Errorf("file writer close error: %w", err)
	}

	return size, nil
}

// Download is Cloud Storageからobjectをダウンロードする
// CMEKとしてBucket Default Keyを指定しているので、コード上はただダウンロードしてるだけ
func (s *CMEKService) Download(ctx context.Context, bucketName string, objectName string) (data []byte, attrs *storage.ObjectAttrs, err error) {
	ctx = trace.StartSpan(ctx, "encryption/cmek/download")
	defer trace.EndSpan(ctx, err)

	rc, attrs, err := s.NewDownloader(ctx, bucketName, objectName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed object.NewReader: %w", err)
	}
	defer func() {
		if err := rc.Close(); err != nil {
			// noop
		}
	}()

	data, err = ioutil.ReadAll(rc)
	if err != nil {
		return nil, nil, fmt.Errorf("failed object.Read: %w", err)
	}
	return data, attrs, nil
}

// Download is Cloud Storageからobjectをダウンロードする
// CMEKとしてBucket Default Keyを指定しているので、コード上はただダウンロードしてるだけ
func (s *CMEKService) NewDownloader(ctx context.Context, bucketName string, objectName string) (w io.ReadCloser, attrs *storage.ObjectAttrs, err error) {
	ctx = trace.StartSpan(ctx, "encryption/cmek/newDownloader")
	defer trace.EndSpan(ctx, err)

	obj := s.gcs.Bucket(bucketName).Object(objectName)
	attrs, err = obj.Attrs(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed read object.Attrs: %w", err)
	}
	rc, err := obj.NewReader(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed object.NewReader: %w", err)
	}

	return rc, attrs, nil
}
```

Bucket全体ではなくObjectごとにKeyを指定することもできます。

``` Go
// UploadWithKey is Cloud Storageに任意のCloud KMS Keyを利用して、fileをアップロードする
// keyName format: "projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s
func (s *CMEKService) UploadWithKey(ctx context.Context, keyName string, bucketName string, objectName string, file []byte) (size int, err error) {
	ctx = trace.StartSpan(ctx, "encryption/cmek/uploadWithKey")
	defer trace.EndSpan(ctx, err)

	obj := s.gcs.Bucket(bucketName).Object(objectName)
	w := obj.NewWriter(ctx)
	w.KMSKeyName = keyName

	size, err = w.Write(file)
	if err != nil {
		return 0, fmt.Errorf("failed gcs.write: %w", err)
	}

	if err := w.Close(); err != nil {
		return size, fmt.Errorf("file writer close error: %w", err)
	}

	return size, nil
}
```

#### ReEncrypt

なんらかの理由で、すでに保存されているObjectを、再度新しいKeyで暗号化したい場合は、Cloud KMSのKeyをRotateし、再度書き込みます。
[Objects: copy](https://cloud.google.com/storage/docs/json_api/v1/objects/copy) を利用すれば、Cloud Storage側で簡単に行うことができます。

``` Go
// ReEncrypt is KeyをRotateした後に、新しいKeyでEncryptし直す時に利用する
// Bucket Default Keyとして設定しているKeyをRotationした後、実行することを想定しているので、実際やっていることはobjectを同じPathにCopyしているだけ
func (s *CMEKService) ReEncrypt(ctx context.Context, bucketName string, objectName string) (err error) {
	ctx = trace.StartSpan(ctx, "encryption/cmek/reEncrypt")
	defer trace.EndSpan(ctx, err)

	src := s.gcs.Bucket(bucketName).Object(objectName)

	// 同じObject PathにCopyする
	// Object Pathが同一でも実際には別のObjectになるので、Copyが成功すれば新しいObjectが返されるようになり、Copy中およびCopyが失敗した場合は元のObjectが返される状態が維持される
	copier := s.gcs.Bucket(bucketName).Object(objectName).CopierFrom(src)
	_, err = copier.Run(ctx)
	if err != nil {
		return err
	}
	return nil
}
```