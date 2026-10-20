package secretbox

import "testing"

func TestEncryptDecryptRoundtrip(t *testing.T) {
	cases := []string{
		"simple-password",
		"p@$$w0rd 'with' \"quotes\" \nnewline",
		"-----BEGIN OPENSSH PRIVATE KEY-----\nabc\ndef-----END-----",
	}
	for _, plain := range cases {
		cipherText, err := Encrypt("test-key", plain)
		if err != nil {
			t.Fatalf("encrypt %q: %v", plain, err)
		}
		if cipherText == plain {
			t.Fatalf("密文不能等于明文")
		}
		got, err := Decrypt("test-key", cipherText)
		if err != nil {
			t.Fatalf("decrypt %q: %v", plain, err)
		}
		if got != plain {
			t.Fatalf("roundtrip 失败: got %q want %q", got, plain)
		}
	}
}

func TestDecryptWrongKey(t *testing.T) {
	cipherText, err := Encrypt("key-a", "secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if _, err := Decrypt("key-b", cipherText); err == nil {
		t.Fatalf("密钥不符时应当解密失败")
	}
}

func TestEmptyKeyRejected(t *testing.T) {
	if _, err := Encrypt("", "secret"); err == nil {
		t.Fatalf("空密钥应当报错")
	}
	if _, err := Decrypt("", "anything"); err == nil {
		t.Fatalf("空密钥应当报错")
	}
}

func TestDecryptCorruptInput(t *testing.T) {
	if _, err := Decrypt("key", "not-base64!!!"); err == nil {
		t.Fatalf("非法 base64 应当报错")
	}
	if _, err := Decrypt("key", "YWJj"); err == nil {
		t.Fatalf("过短密文应当报错")
	}
}
