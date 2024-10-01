resource "yandex_kms_asymmetric_signature_key" "your-key" {
  folder_id = "your-folder-id"
  name      = "asymmetric-signature-key-name"
}

resource "yandex_kms_asymmetric_signature_key_iam_binding" "viewer" {
  asymmetric_signaturen_key_id = yandex_kms_asymmetric_signature_key.your-key.id
  role                         = "viewer"

  members = [
    "userAccount:foo_user_id",
  ]
}