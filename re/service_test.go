// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package re_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/0x6flab/namegenerator"
	"github.com/absmach/magistrala/internal/testsutil"
	"github.com/absmach/magistrala/re"
	"github.com/absmach/magistrala/re/mocks"
	readmocks "github.com/absmach/magistrala/readers/mocks"
	"github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	repoerr "github.com/absmach/supermq/pkg/errors/repository"
	svcerr "github.com/absmach/supermq/pkg/errors/service"
	"github.com/absmach/supermq/pkg/messaging"
	pubsubmocks "github.com/absmach/supermq/pkg/messaging/mocks"
	"github.com/absmach/supermq/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	namegen      = namegenerator.NewGenerator()
	userID       = testsutil.GenerateUUID(&testing.T{})
	domainID     = testsutil.GenerateUUID(&testing.T{})
	ruleName     = namegen.Generate()
	ruleID       = testsutil.GenerateUUID(&testing.T{})
	inputChannel = "test.channel"
	schedule     = re.Schedule{
		StartDateTime:   time.Now().Add(-time.Hour),
		Recurring:       re.Daily,
		RecurringPeriod: 1,
		Time:            time.Now().Add(-time.Hour),
	}
	reportName = namegen.Generate()
	rptConfig  = re.ReportConfig{
		ID:        testsutil.GenerateUUID(&testing.T{}),
		Name:      reportName,
		DomainID:  domainID,
		Status:    re.EnabledStatus,
		Schedule:  schedule,
		CreatedBy: userID,
		UpdatedBy: userID,
		UpdatedAt: time.Now(),
	}
)

func newService(t *testing.T, runInfo chan re.RunInfo) (re.Service, *mocks.Repository, *pubsubmocks.PubSub, *mocks.Ticker) {
	repo := new(mocks.Repository)
	mockTicker := new(mocks.Ticker)
	idProvider := uuid.NewMock()
	pubsub := pubsubmocks.NewPubSub(t)
	readersSvc := new(readmocks.ReadersServiceClient)
	e := new(mocks.Emailer)
	return re.NewService(repo, runInfo, idProvider, pubsub, pubsub, pubsub, mockTicker, e, readersSvc), repo, pubsub, mockTicker
}

func TestAddRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))
	ruleName := namegen.Generate()
	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		res     re.Rule
		err     error
	}{
		{
			desc: "Add rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
			},
			res: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: nil,
		},
		{
			desc: "Add rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("AddRule", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.AddRule(context.Background(), tc.session, tc.rule)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.NotEmpty(t, res.ID, "expected non-empty result in ID")
				assert.Equal(t, tc.rule.Name, res.Name)
				assert.Equal(t, tc.rule.Schedule, res.Schedule)
			}
			defer repoCall.Unset()
		})
	}
}

func TestViewRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		id      string
		res     re.Rule
		err     error
	}{
		{
			desc: "view rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id: ruleID,
			res: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: nil,
		},
		{
			desc: "view rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  ruleID,
			err: svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ViewRule", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ViewRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestUpdateRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	newName := namegen.Generate()
	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		rule    re.Rule
		res     re.Rule
		err     error
	}{
		{
			desc: "update rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         newName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			res: re.Rule{
				Name:         newName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
				UpdatedAt: now,
				UpdatedBy: userID,
			},
			err: nil,
		},
		{
			desc: "update rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			rule: re.Rule{
				Name:         ruleName,
				ID:           ruleID,
				InputChannel: inputChannel,
				Schedule: re.Schedule{
					Recurring:       re.Daily,
					RecurringPeriod: 1,
					Time:            now,
				},
				Status:    re.EnabledStatus,
				CreatedBy: userID,
				DomainID:  domainID,
			},
			err: svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRule", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.UpdateRule(context.Background(), tc.session, tc.rule)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestListRules(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))
	numRules := 50
	now := time.Now().Add(time.Hour)
	var rules []re.Rule
	for i := 0; i < numRules; i++ {
		r := re.Rule{
			ID:        testsutil.GenerateUUID(t),
			Name:      namegen.Generate(),
			DomainID:  domainID,
			Status:    re.EnabledStatus,
			CreatedAt: now,
			CreatedBy: userID,
			Schedule: re.Schedule{
				Recurring:       re.Daily,
				Time:            now.Add(1 * time.Hour),
				RecurringPeriod: 1,
				StartDateTime:   now.Add(-1 * time.Hour),
			},
		}
		rules = append(rules, r)
	}

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta re.PageMeta
		res      re.Page
		err      error
	}{
		{
			desc: "list rules successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			res: re.Page{
				Total:  uint64(numRules),
				Offset: 0,
				Limit:  10,
				Rules:  rules[0:10],
			},
			err: nil,
		},
		{
			desc: "list rules successfully with limit",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{
				Limit: 100,
			},
			res: re.Page{
				Total:  uint64(numRules),
				Offset: 0,
				Limit:  100,
				Rules:  rules[0:numRules],
			},
			err: nil,
		},
		{
			desc: "list rules successfully with offset",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{
				Offset: 20,
				Limit:  10,
			},
			res: re.Page{
				Total:  uint64(numRules),
				Offset: 20,
				Limit:  10,
				Rules:  rules[20:30],
			},
			err: nil,
		},
		{
			desc: "list rules with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ListRules(context.Background(), tc.session, tc.pageMeta)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestRemoveRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		err     error
	}{
		{
			desc: "remove rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  ruleID,
			err: nil,
		},
		{
			desc: "remove rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  ruleID,
			err: svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RemoveRule", mock.Anything, mock.Anything).Return(tc.err)
			err := svc.RemoveRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			defer repoCall.Unset()
		})
	}
}

func TestEnableRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	now := time.Now()

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  re.Status
		res     re.Rule
		err     error
	}{
		{
			desc: "enable rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.EnabledStatus,
			res: re.Rule{
				ID:           ruleID,
				Name:         ruleName,
				DomainID:     domainID,
				InputChannel: inputChannel,
				Status:       re.EnabledStatus,
				Schedule:     schedule,
				UpdatedBy:    userID,
				UpdatedAt:    now,
			},
			err: nil,
		},
		{
			desc: "enable rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.EnabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRuleStatus", context.Background(), mock.Anything).Return(tc.res, tc.err)
			res, err := svc.EnableRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestDisableRule(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	now := time.Now()

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  re.Status
		res     re.Rule
		err     error
	}{
		{
			desc: "disable rule successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.DisabledStatus,
			res: re.Rule{
				ID:           ruleID,
				Name:         ruleName,
				DomainID:     domainID,
				InputChannel: inputChannel,
				Status:       re.DisabledStatus,
				Schedule:     schedule,
				UpdatedBy:    userID,
				UpdatedAt:    now,
			},
			err: nil,
		},
		{
			desc: "disable rule with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     ruleID,
			status: re.DisabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateRuleStatus", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.DisableRule(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestHandle(t *testing.T) {
	svc, repo, pubmocks, _ := newService(t, make(chan re.RunInfo))
	now := time.Now()
	scheduled := false
	cases := []struct {
		desc       string
		message    *messaging.Message
		page       re.Page
		listErr    error
		publishErr error
		expectErr  bool
	}{
		{
			desc: "consume message with empty rules",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
			},
			page: re.Page{
				Rules: []re.Rule{},
			},
			listErr: nil,
		},
		{
			desc: "consume message with rules",
			message: &messaging.Message{
				Channel: inputChannel,
				Created: now.Unix(),
			},
			page: re.Page{
				Rules: []re.Rule{
					{
						ID:           testsutil.GenerateUUID(t),
						Name:         namegen.Generate(),
						InputChannel: inputChannel,
						Status:       re.EnabledStatus,
						Logic: re.Script{
							Type: re.ScriptType(0),
						},
						OutputChannel: "output.channel",
						Schedule:      schedule,
					},
				},
			},
			listErr: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			var err error

			repoCall := repo.On("ListRules", mock.Anything, re.PageMeta{Domain: tc.message.Domain, InputChannel: tc.message.Channel, Scheduled: &scheduled}).Return(tc.page, tc.listErr).Run(func(args mock.Arguments) {
				if tc.listErr != nil {
					err = tc.listErr
				}
			})
			repoCall1 := pubmocks.On("Publish", mock.Anything, mock.Anything, mock.Anything).Return(tc.publishErr)
			repoCall2 := repo.On("ListReportsConfig", mock.Anything, mock.Anything).Return(re.ReportConfigPage{}, nil)

			err = svc.Handle(tc.message)
			assert.Nil(t, err)

			assert.True(t, errors.Contains(err, tc.listErr), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.listErr, err))

			repoCall.Unset()
			repoCall1.Unset()
			repoCall2.Unset()
		})
	}
}

func TestAddReportConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		cfg     re.ReportConfig
		res     re.ReportConfig
		err     error
	}{
		{
			desc: "Add report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: re.ReportConfig{
				Name:     reportName,
				Schedule: schedule,
			},
			res: rptConfig,
			err: nil,
		},
		{
			desc: "Add report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: re.ReportConfig{
				Name:     reportName,
				Schedule: schedule,
			},
			err: repoerr.ErrCreateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("AddReportConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.AddReportConfig(context.Background(), tc.session, tc.cfg)
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.NotEmpty(t, res.ID, "expected non-empty result in ID")
				assert.Equal(t, tc.cfg.Name, res.Name)
				assert.Equal(t, tc.cfg.Schedule, res.Schedule)
			}
			defer repoCall.Unset()
		})
	}
}

func TestViewReportConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		res     re.ReportConfig
		err     error
	}{
		{
			desc: "view report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			res: rptConfig,
			err: nil,
		},
		{
			desc: "view report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			err: svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ViewReportConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ViewReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestUpdateReportConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	newName := namegen.Generate()
	now := time.Now().Add(time.Hour)
	cases := []struct {
		desc    string
		session authn.Session
		cfg     re.ReportConfig
		res     re.ReportConfig
		err     error
	}{
		{
			desc: "update report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: re.ReportConfig{
				Name:     newName,
				ID:       rptConfig.ID,
				Schedule: schedule,
			},
			res: re.ReportConfig{
				Name:      newName,
				ID:        rptConfig.ID,
				DomainID:  rptConfig.DomainID,
				Status:    rptConfig.Status,
				Schedule:  rptConfig.Schedule,
				UpdatedAt: now,
				UpdatedBy: userID,
			},
			err: nil,
		},
		{
			desc: "update report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			cfg: re.ReportConfig{
				Name:     rptConfig.Name,
				ID:       rptConfig.ID,
				Schedule: schedule,
			},
			err: svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateReportConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.UpdateReportConfig(context.Background(), tc.session, tc.cfg)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestListReportsConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))
	numConfigs := 50
	now := time.Now().Add(time.Hour)
	var configs []re.ReportConfig
	for i := 0; i < numConfigs; i++ {
		c := re.ReportConfig{
			ID:        testsutil.GenerateUUID(t),
			Name:      namegen.Generate(),
			DomainID:  domainID,
			Status:    re.EnabledStatus,
			CreatedAt: now,
			CreatedBy: userID,
			Schedule:  schedule,
		}
		configs = append(configs, c)
	}

	cases := []struct {
		desc     string
		session  authn.Session
		pageMeta re.PageMeta
		res      re.ReportConfigPage
		err      error
	}{
		{
			desc: "list report configs successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			res: re.ReportConfigPage{
				PageMeta: re.PageMeta{
					Total:  uint64(numConfigs),
					Offset: 0,
					Limit:  10,
				},
				ReportConfigs: configs[0:10],
			},
			err: nil,
		},
		{
			desc: "list report configs successfully with limit",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{
				Limit: 100,
			},
			res: re.ReportConfigPage{
				PageMeta: re.PageMeta{
					Total:  uint64(numConfigs),
					Offset: 0,
					Limit:  100,
				},
				ReportConfigs: configs[0:numConfigs],
			},
			err: nil,
		},
		{
			desc: "list report configs successfully with offset",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{
				Offset: 20,
				Limit:  10,
			},
			res: re.ReportConfigPage{
				PageMeta: re.PageMeta{
					Total:  uint64(numConfigs),
					Offset: 20,
					Limit:  10,
				},
				ReportConfigs: configs[20:30],
			},
			err: nil,
		},
		{
			desc: "list report configs with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			pageMeta: re.PageMeta{},
			err:      svcerr.ErrViewEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListReportsConfig", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.ListReportsConfig(context.Background(), tc.session, tc.pageMeta)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestRemoveReportConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		err     error
	}{
		{
			desc: "remove report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			err: nil,
		},
		{
			desc: "remove report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:  rptConfig.ID,
			err: svcerr.ErrRemoveEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("RemoveReportConfig", mock.Anything, mock.Anything).Return(tc.err)
			err := svc.RemoveReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			defer repoCall.Unset()
		})
	}
}

func TestEnableReportConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  re.Status
		res     re.ReportConfig
		err     error
	}{
		{
			desc: "enable report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: re.EnabledStatus,
			res:    rptConfig,
			err:    nil,
		},
		{
			desc: "enable report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: re.EnabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateReportConfigStatus", context.Background(), mock.Anything).Return(tc.res, tc.err)
			res, err := svc.EnableReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestDisableReportConfig(t *testing.T) {
	svc, repo, _, _ := newService(t, make(chan re.RunInfo))

	cases := []struct {
		desc    string
		session authn.Session
		id      string
		status  re.Status
		res     re.ReportConfig
		err     error
	}{
		{
			desc: "disable report config successfully",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: re.DisabledStatus,
			res: re.ReportConfig{
				ID:        rptConfig.ID,
				Name:      rptConfig.Name,
				DomainID:  rptConfig.DomainID,
				Status:    re.DisabledStatus,
				Schedule:  schedule,
				UpdatedBy: userID,
				UpdatedAt: time.Now(),
			},
			err: nil,
		},
		{
			desc: "disable report config with failed repo",
			session: authn.Session{
				UserID:   userID,
				DomainID: domainID,
			},
			id:     rptConfig.ID,
			status: re.DisabledStatus,
			err:    svcerr.ErrUpdateEntity,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("UpdateReportConfigStatus", mock.Anything, mock.Anything).Return(tc.res, tc.err)
			res, err := svc.DisableReportConfig(context.Background(), tc.session, tc.id)

			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("%s: expected %s got %s\n", tc.desc, tc.err, err))
			if err == nil {
				assert.Equal(t, tc.res, res)
			}
			defer repoCall.Unset()
		})
	}
}

func TestStartScheduler(t *testing.T) {
	now := time.Now().Truncate(time.Minute)
	ri := make(chan re.RunInfo)
	svc, repo, _, ticker := newService(t, ri)

	ctxCases := []struct {
		desc     string
		err      error
		pageMeta re.PageMeta
		page     re.Page
		listErr  error
		setupCtx func() (context.Context, context.CancelFunc)
	}{
		{
			desc: "start scheduler with canceled context",
			err:  context.Canceled,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
		},
		{
			desc: "start scheduler with timeout",
			err:  context.DeadlineExceeded,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Millisecond)
			},
		},
		{
			desc: "start scheduler with deadline exceeded",
			err:  context.DeadlineExceeded,
			pageMeta: re.PageMeta{
				Status:          re.EnabledStatus,
				ScheduledBefore: &now,
			},
			page: re.Page{},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(time.Millisecond))
			},
		},
	}

	for _, tc := range ctxCases {
		t.Run(tc.desc, func(t *testing.T) {
			repoCall := repo.On("ListRules", mock.Anything, mock.Anything).Return(tc.page, tc.listErr)
			repoCall1 := repo.On("ListReportsConfig", mock.Anything, mock.Anything).Return(re.ReportConfigPage{}, nil)
			tickChan := make(chan time.Time)
			tickCall := ticker.On("Tick").Return((<-chan time.Time)(tickChan))
			tickCall1 := ticker.On("Stop").Return()
			ctx, cancel := tc.setupCtx()
			defer cancel()
			errc := make(chan error)

			go func() {
				errc <- svc.StartScheduler(ctx)
			}()

			err := <-errc
			assert.True(t, errors.Contains(err, tc.err), fmt.Sprintf("expected error %v but got %v", tc.err, err))
			repoCall.Unset()
			repoCall1.Unset()
			tickCall.Unset()
			tickCall1.Unset()
		})
	}
}
