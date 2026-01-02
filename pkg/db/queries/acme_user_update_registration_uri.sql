-- name: UpdateAcmeUserRegistrationURI :exec
UPDATE acme_users SET registration_uri = ? WHERE id = ?;
