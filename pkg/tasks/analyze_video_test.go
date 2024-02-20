package tasks

import (
	"testing"
	"time"
)

func Test_parseStringToFloat(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		{
			name: "div by 1",
			args: args{
				"100",
			},
			want:    100,
			wantErr: false,
		},
		{
			name: "div by 2",
			args: args{
				"100/2",
			},
			want:    50,
			wantErr: false,
		},
		{
			name: "div by 1001",
			args: args{
				"24000/1001",
			},
			want:    float64(24000) / float64(1001),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseStringToFloat(tt.args.s)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseStringToFloat() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseStringToFloat() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAnalyzeResults_SetDurationFromFramerateString(t *testing.T) {
	type fields struct {
		Duration    time.Duration
		TotalFrames int
	}
	type args struct {
		framerate string
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		want     time.Duration
		wantErr  bool
		rounding time.Duration
	}{
		{
			"~1 second",
			fields{
				TotalFrames: 24,
			},
			args{framerate: "24"},
			time.Second,
			false,
			0,
		},
		{
			"~10 minutes",
			fields{
				TotalFrames: 14386,
			},
			args{framerate: "24"},
			10 * time.Minute,
			false,
			time.Minute,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AnalyzeResults{
				Duration:    tt.fields.Duration,
				TotalFrames: tt.fields.TotalFrames,
			}
			if err := r.SetDurationFromFramerateString(tt.args.framerate); (err != nil) != tt.wantErr {
				t.Errorf("SetDurationFromFramerateString() error = %v, wantErr %v", err, tt.wantErr)
			}
			if r.Duration.Round(tt.rounding) != tt.want {
				t.Errorf("r.Duration got = %v, want %v", r.Duration.Round(tt.rounding), tt.want)
			}
		})
	}
}
