package handlers

import (
	"bufio"
	"testing"
)

func Test_parseAndValidateRESPRequest(t *testing.T) {
	type args struct {
		reader *bufio.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAndValidateRESPRequest(tt.args.reader)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAndValidateRESPRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseAndValidateRESPRequest() got = %v, want %v", got, tt.want)
			}
		})
	}
}
