package client

import (
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ingelectus/stomp"
	"github.com/ingelectus/stomp/frame"
)

const (
	// Maximum permitted heart-beat timeout, about 11.5 days.
	// Any client CONNECT frame with a larger value than this
	// will be rejected.
	maxHeartBeat = 999999999
)

var (
	// Regexp for heart-beat header value
	heartBeatRegexp = regexp.MustCompile("^[0-9]{1,9},[0-9]{1,9}$")
)

// Determine the most acceptable version based on the accept-version
// header of a CONNECT or STOMP frame.
//
// Returns stomp.V10 if a CONNECT frame and the accept-version header is missing.
//
// Returns an error if the frame is not a CONNECT or STOMP frame, or
// if the accept-header is malformed or does not contain any compatible
// version numbers. Also returns an error if the accept-header is missing
// for a STOMP frame.
//
// Otherwise, returns the highest compatible version specified in the
// accept-version header. Compatible versions are V1_0, V1_1 and V1_2.
func determineVersion(f *frame.Frame) (version stomp.Version, err error) {
	// frame can be CONNECT or STOMP with slightly different
	// handling of accept-verion for each
	isConnect := f.Command == frame.CONNECT

	if !isConnect && f.Command != frame.STOMP {
		err = notConnectFrame
		return
	}

	// start with an error, and remove if successful
	err = unknownVersion

	if acceptVersion, ok := f.Header.Contains(frame.AcceptVersion); ok {
		// sort the versions so that the latest version comes last
		versions := strings.Split(acceptVersion, ",")
		sort.Strings(versions)
		for _, v := range versions {
			switch stomp.Version(v) {
			case stomp.V10:
				version = stomp.V10
				err = nil
			case stomp.V11:
				version = stomp.V11
				err = nil
			case stomp.V12:
				version = stomp.V12
				err = nil
			}
		}
	} else {
		// CONNECT frames can be missing the accept-version header,
		// we assume V1.0 in this case. STOMP frames were introduced
		// in V1.1, so they must have an accept-version header.
		if isConnect {
			// no "accept-version" header, so we assume 1.0
			version = stomp.V10
			err = nil
		} else {
			err = missingHeader(frame.AcceptVersion)
		}
	}
	return
}

// Determine the heart-beat values in a CONNECT or STOMP frame.
//
// Returns 0,0 if the heart-beat header is missing. Otherwise
// returns the cx and cy values in the frame.
//
// Returns an error if the heart-beat header is malformed, or if
// the frame is not a CONNECT or STOMP frame. In this implementation,
// a heart-beat header is considered malformed if either cx or cy
// is greater than MaxHeartBeat.
func getHeartBeat(f *frame.Frame) (cx, cy int, err error) {
	if f.Command != frame.CONNECT &&
		f.Command != frame.STOMP &&
		f.Command != frame.CONNECTED {
		err = invalidOperationForFrame
		return
	}
	if heartBeat, ok := f.Header.Contains(frame.HeartBeat); ok {
		if !heartBeatRegexp.MatchString(heartBeat) {
			err = invalidHeartBeat
			return
		}

		// no error checking here because we are confident
		// that everything will work because the regexp matches.
		slice := strings.Split(heartBeat, ",")
		value1, _ := strconv.ParseUint(slice[0], 10, 32)
		value2, _ := strconv.ParseUint(slice[1], 10, 32)
		cx = int(value1)
		cy = int(value2)
	} else {
		// heart-beat header not present
		// this else clause is not necessary, but
		// included for clarity.
		cx = 0
		cy = 0
	}
	return
}
