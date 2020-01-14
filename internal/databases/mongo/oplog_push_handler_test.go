package mongo

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/wal-g/wal-g/internal/databases/mongo/mocks"
	"github.com/wal-g/wal-g/internal/databases/mongo/oplog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type fromFetcherReturn struct {
	outChan chan oplog.Record
	errChan chan error
	err     error
}

type validatorReturn struct {
	outChan chan oplog.Record
	errChan chan error
	err     error
}

type applierReturn struct {
	errChan chan error
	err     error
}

type oplogPushArgs struct {
	ctx   context.Context
	since oplog.Timestamp
	wg    *sync.WaitGroup

	fromFetcherReturn *fromFetcherReturn
	validatorReturn   *validatorReturn
	applierReturn     *applierReturn
}

type oplogPushMocks struct {
	fetcher   *mocks.FromFetcher
	validator *mocks.Validator
	applier   *mocks.Applier
}

func (tm *oplogPushMocks) AssertExpectations(t *testing.T) {
	if tm.fetcher != nil {
		tm.fetcher.AssertExpectations(t)
	}
	if tm.validator != nil {
		tm.validator.AssertExpectations(t)
	}
	if tm.applier != nil {
		tm.applier.AssertExpectations(t)
	}
}

func buildTestArgs() oplogPushArgs {
	return oplogPushArgs{
		ctx:   context.TODO(),
		since: oplog.Timestamp{TS: 1579021614, Inc: 15},
		wg:    &sync.WaitGroup{},

		fromFetcherReturn: &fromFetcherReturn{make(chan oplog.Record), make(chan error), nil},
		validatorReturn:   &validatorReturn{make(chan oplog.Record), make(chan error), nil},
		applierReturn:     &applierReturn{make(chan error), nil},
	}
}

func prepareOplogPushMocks(args oplogPushArgs, mocks oplogPushMocks) {
	if mocks.fetcher != nil {
		mocks.fetcher.On("OplogFrom", mock.Anything, args.since, args.wg).
			Return(args.fromFetcherReturn.outChan, args.fromFetcherReturn.errChan, args.fromFetcherReturn.err)
	}
	if mocks.validator != nil {
		mocks.validator.On("Validate", mock.Anything, args.fromFetcherReturn.outChan, args.wg).
			Return(args.validatorReturn.outChan, args.validatorReturn.errChan, args.validatorReturn.err)
	}
	if mocks.applier != nil {
		mocks.applier.On("Apply", mock.Anything, args.validatorReturn.outChan, args.wg).
			Return(args.applierReturn.errChan, args.applierReturn.err)
	}
}

func TestHandleOplogPush(t *testing.T) {
	tests := []struct {
		name        string
		args        oplogPushArgs
		mocks       oplogPushMocks
		failErrRet  func(args oplogPushArgs)
		failErrChan func(args oplogPushArgs)
		expectedErr error
	}{
		{
			name:        "fetcher call returns error",
			args:        buildTestArgs(),
			mocks:       oplogPushMocks{&mocks.FromFetcher{}, nil, nil},
			failErrRet:  func(args oplogPushArgs) { args.fromFetcherReturn.err = fmt.Errorf("fetcher ret err") },
			expectedErr: fmt.Errorf("fetcher ret err"),
		},
		{
			name:        "validator call returns error",
			args:        buildTestArgs(),
			mocks:       oplogPushMocks{&mocks.FromFetcher{}, &mocks.Validator{}, nil},
			failErrRet:  func(args oplogPushArgs) { args.validatorReturn.err = fmt.Errorf("validator ret err") },
			expectedErr: fmt.Errorf("validator ret err"),
		},
		{
			name:        "applier call returns error",
			args:        buildTestArgs(),
			mocks:       oplogPushMocks{&mocks.FromFetcher{}, &mocks.Validator{}, &mocks.Applier{}},
			failErrRet:  func(args oplogPushArgs) { args.applierReturn.err = fmt.Errorf("applier ret err") },
			expectedErr: fmt.Errorf("applier ret err"),
		},
		{
			name:        "fetcher returns error via error channel",
			args:        buildTestArgs(),
			mocks:       oplogPushMocks{&mocks.FromFetcher{}, &mocks.Validator{}, &mocks.Applier{}},
			failErrChan: func(args oplogPushArgs) { args.fromFetcherReturn.errChan <- fmt.Errorf("fetcher chan err") },
			expectedErr: fmt.Errorf("fetcher chan err"),
		},
		{
			name:        "validator returns error via error channel",
			args:        buildTestArgs(),
			mocks:       oplogPushMocks{&mocks.FromFetcher{}, &mocks.Validator{}, &mocks.Applier{}},
			failErrChan: func(args oplogPushArgs) { args.fromFetcherReturn.errChan <- fmt.Errorf("validator chan err") },
			expectedErr: fmt.Errorf("validator chan err"),
		},
		{
			name:        "applier returns error via error channel",
			args:        buildTestArgs(),
			mocks:       oplogPushMocks{&mocks.FromFetcher{}, &mocks.Validator{}, &mocks.Applier{}},
			failErrChan: func(args oplogPushArgs) { args.fromFetcherReturn.errChan <- fmt.Errorf("applier chan err") },
			expectedErr: fmt.Errorf("applier chan err"),
		},
	}

	for _, tc := range tests {
		if tc.failErrRet != nil {
			tc.failErrRet(tc.args)
		}
		if tc.failErrChan != nil {
			go tc.failErrChan(tc.args)
		}

		prepareOplogPushMocks(tc.args, tc.mocks)
		err := HandleOplogPush(tc.args.ctx, tc.args.since, tc.mocks.fetcher, tc.mocks.validator, tc.mocks.applier)
		if tc.expectedErr != nil {
			assert.EqualError(t, err, tc.expectedErr.Error())
		} else {
			assert.Nil(t, err)
		}

		tc.mocks.AssertExpectations(t)
	}
}
