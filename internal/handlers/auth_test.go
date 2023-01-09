package handlers

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestHandler_handleCookie(t *testing.T) {
	type fields struct {
		Mux     *chi.Mux
		storage Storager
		user    *user
	}
	type args struct {
		next http.HandlerFunc
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   http.HandlerFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Mux:     tt.fields.Mux,
				storage: tt.fields.storage,
				user:    tt.fields.user,
			}
			assert.Equalf(t, tt.want, h.handleCookie(tt.args.next), "handleCookie(%v)", tt.args.next)
		})
	}
}

func Test_newUser(t *testing.T) {
	tests := []struct {
		name    string
		want    *user
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newUser()
			if !tt.wantErr(t, err, fmt.Sprintf("newUser()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "newUser()")
		})
	}
}

func Test_user_get(t *testing.T) {
	type fields struct {
		id     string
		sign   []byte
		cookie string
	}
	type args struct {
		cookie string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &user{
				id:     tt.fields.id,
				sign:   tt.fields.sign,
				cookie: tt.fields.cookie,
			}
			tt.wantErr(t, u.get(tt.args.cookie), fmt.Sprintf("get(%v)", tt.args.cookie))
		})
	}
}

func Test_user_signId(t *testing.T) {
	type fields struct {
		id     string
		sign   []byte
		cookie string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr assert.ErrorAssertionFunc
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := &user{
				id:     tt.fields.id,
				sign:   tt.fields.sign,
				cookie: tt.fields.cookie,
			}
			got, err := signId(u.id)
			if !tt.wantErr(t, err, fmt.Sprintf("signId()")) {
				return
			}
			assert.Equalf(t, tt.want, got, "signId()")
		})
	}
}
