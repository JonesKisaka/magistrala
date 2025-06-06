// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/absmach/magistrala/re"
	"github.com/absmach/supermq"
	api "github.com/absmach/supermq/api/http"
	apiutil "github.com/absmach/supermq/api/http/util"
	mgauthn "github.com/absmach/supermq/pkg/authn"
	"github.com/absmach/supermq/pkg/errors"
	"github.com/go-chi/chi/v5"
	kithttp "github.com/go-kit/kit/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	ruleIdKey        = "ruleID"
	reportIdKey      = "reportID"
	inputChannelKey  = "input_channel"
	outputChannelKey = "output_channel"
	statusKey        = "status"
	actionKey        = "action"
	defAction        = "view"
)

// MakeHandler creates an HTTP handler for the service endpoints.
func MakeHandler(svc re.Service, authn mgauthn.Authentication, mux *chi.Mux, logger *slog.Logger, instanceID string) http.Handler {
	opts := []kithttp.ServerOption{
		kithttp.ServerErrorEncoder(apiutil.LoggingErrorEncoder(logger, api.EncodeError)),
	}
	mux.Group(func(r chi.Router) {
		r.Use(api.AuthenticateMiddleware(authn, true))
		r.Route("/{domainID}", func(r chi.Router) {
			r.Route("/rules", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					addRuleEndpoint(svc),
					decodeAddRuleRequest,
					api.EncodeResponse,
					opts...,
				), "create_rule").ServeHTTP)

				r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
					listRulesEndpoint(svc),
					decodeListRulesRequest,
					api.EncodeResponse,
					opts...,
				), "list_rules").ServeHTTP)

				r.Route("/{ruleID}", func(r chi.Router) {
					r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
						viewRuleEndpoint(svc),
						decodeViewRuleRequest,
						api.EncodeResponse,
						opts...,
					), "view_rule").ServeHTTP)

					r.Patch("/", otelhttp.NewHandler(kithttp.NewServer(
						updateRuleEndpoint(svc),
						decodeUpdateRuleRequest,
						api.EncodeResponse,
						opts...,
					), "update_rule").ServeHTTP)

					r.Patch("/schedule", otelhttp.NewHandler(kithttp.NewServer(
						updateRuleScheduleEndpoint(svc),
						decodeUpdateRuleScheduleRequest,
						api.EncodeResponse,
						opts...,
					), "update_rule_scheduler").ServeHTTP)

					r.Delete("/", otelhttp.NewHandler(kithttp.NewServer(
						deleteRuleEndpoint(svc),
						decodeDeleteRuleRequest,
						api.EncodeResponse,
						opts...,
					), "delete_rule").ServeHTTP)

					r.Post("/enable", otelhttp.NewHandler(kithttp.NewServer(
						enableRuleEndpoint(svc),
						decodeUpdateRuleStatusRequest,
						api.EncodeResponse,
						opts...,
					), "enable_rule").ServeHTTP)

					r.Post("/disable", otelhttp.NewHandler(kithttp.NewServer(
						disableRuleEndpoint(svc),
						decodeUpdateRuleStatusRequest,
						api.EncodeResponse,
						opts...,
					), "disable_rule").ServeHTTP)
				})
			})
			r.Route("/reports", func(r chi.Router) {
				r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
					generateReportEndpoint(svc),
					decodeGenerateReportRequest,
					encodeFileDownloadResponse,
					opts...,
				), "generate_report").ServeHTTP)

				r.Route("/configs", func(r chi.Router) {
					r.Post("/", otelhttp.NewHandler(kithttp.NewServer(
						addReportConfigEndpoint(svc),
						decodeAddReportConfigRequest,
						api.EncodeResponse,
						opts...,
					), "add_report_config").ServeHTTP)

					r.Get("/{reportID}", otelhttp.NewHandler(kithttp.NewServer(
						viewReportConfigEndpoint(svc),
						decodeViewReportConfigRequest,
						api.EncodeResponse,
						opts...,
					), "view_report_config").ServeHTTP)

					r.Patch("/{reportID}", otelhttp.NewHandler(kithttp.NewServer(
						updateReportConfigEndpoint(svc),
						decodeUpdateReportConfigRequest,
						api.EncodeResponse,
						opts...,
					), "update_report_config").ServeHTTP)

					r.Patch("/{reportID}/schedule", otelhttp.NewHandler(kithttp.NewServer(
						updateReportScheduleEndpoint(svc),
						decodeUpdateReportScheduleRequest,
						api.EncodeResponse,
						opts...,
					), "update_report_scheduler").ServeHTTP)

					r.Delete("/{reportID}", otelhttp.NewHandler(kithttp.NewServer(
						deleteReportConfigEndpoint(svc),
						decodeDeleteReportConfigRequest,
						api.EncodeResponse,
						opts...,
					), "delete_report_config").ServeHTTP)

					r.Get("/", otelhttp.NewHandler(kithttp.NewServer(
						listReportsConfigEndpoint(svc),
						decodeListReportsConfigRequest,
						api.EncodeResponse,
						opts...,
					), "list_reports_config").ServeHTTP)

					r.Post("/{reportID}/enable", otelhttp.NewHandler(kithttp.NewServer(
						enableReportConfigEndpoint(svc),
						decodeUpdateReportStatusRequest,
						api.EncodeResponse,
						opts...,
					), "enable_report_config").ServeHTTP)

					r.Post("/{reportID}/disable", otelhttp.NewHandler(kithttp.NewServer(
						disableReportConfigEndpoint(svc),
						decodeUpdateReportStatusRequest,
						api.EncodeResponse,
						opts...,
					), "disable_report_config").ServeHTTP)
				})
			})
		})
	})

	mux.Get("/health", supermq.Health("rule_engine", instanceID))
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

func decodeAddRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var rule re.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	return addRuleReq{Rule: rule}, nil
}

func decodeViewRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, ruleIdKey)
	return viewRuleReq{id: id}, nil
}

func decodeUpdateRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var rule re.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}
	rule.ID = chi.URLParam(r, ruleIdKey)
	return updateRuleReq{Rule: rule}, nil
}

func decodeUpdateRuleScheduleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateRuleScheduleReq{
		id: chi.URLParam(r, ruleIdKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdateRuleStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := updateRuleStatusReq{
		id: chi.URLParam(r, ruleIdKey),
	}
	return req, nil
}

func decodeListRulesRequest(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	ic, err := apiutil.ReadStringQuery(r, inputChannelKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	oc, err := apiutil.ReadStringQuery(r, outputChannelKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	s, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefStatus)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	dir, err := apiutil.ReadStringQuery(r, api.DirKey, api.DefDir)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := re.ToStatus(s)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	return listRulesReq{
		PageMeta: re.PageMeta{
			Offset:        offset,
			Limit:         limit,
			Name:          name,
			InputChannel:  ic,
			OutputChannel: oc,
			Status:        st,
			Dir:           dir,
		},
	}, nil
}

func decodeDeleteRuleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, ruleIdKey)

	return deleteRuleReq{id: id}, nil
}

func decodeGenerateReportRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	a, err := apiutil.ReadStringQuery(r, actionKey, defAction)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	action, err := re.ToReportAction(a)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}

	req := generateReportReq{
		action: action,
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(err, apiutil.ErrValidation)
	}

	return req, nil
}

func decodeAddReportConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var config re.ReportConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		return nil, errors.Wrap(err, apiutil.ErrValidation)
	}
	return addReportConfigReq{ReportConfig: config}, nil
}

func decodeViewReportConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, reportIdKey)
	return viewReportConfigReq{ID: id}, nil
}

func decodeUpdateReportConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}
	var config re.ReportConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		return nil, errors.Wrap(err, apiutil.ErrValidation)
	}
	config.ID = chi.URLParam(r, reportIdKey)
	return updateReportConfigReq{ReportConfig: config}, nil
}

func decodeUpdateReportScheduleRequest(_ context.Context, r *http.Request) (interface{}, error) {
	if !strings.Contains(r.Header.Get("Content-Type"), api.ContentType) {
		return nil, errors.Wrap(apiutil.ErrValidation, apiutil.ErrUnsupportedContentType)
	}

	req := updateReportScheduleReq{
		id: chi.URLParam(r, reportIdKey),
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, errors.Wrap(errors.ErrMalformedEntity, err))
	}

	return req, nil
}

func decodeUpdateReportStatusRequest(_ context.Context, r *http.Request) (interface{}, error) {
	req := updateReportStatusReq{
		id: chi.URLParam(r, reportIdKey),
	}
	return req, nil
}

func decodeDeleteReportConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	id := chi.URLParam(r, reportIdKey)
	return deleteReportConfigReq{ID: id}, nil
}

func decodeListReportsConfigRequest(_ context.Context, r *http.Request) (interface{}, error) {
	offset, err := apiutil.ReadNumQuery[uint64](r, api.OffsetKey, api.DefOffset)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	limit, err := apiutil.ReadNumQuery[uint64](r, api.LimitKey, api.DefLimit)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	status, err := apiutil.ReadStringQuery(r, api.StatusKey, api.DefStatus)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	st, err := re.ToStatus(status)
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	name, err := apiutil.ReadStringQuery(r, api.NameKey, "")
	if err != nil {
		return nil, errors.Wrap(apiutil.ErrValidation, err)
	}
	return listReportsConfigReq{
		PageMeta: re.PageMeta{
			Offset: offset,
			Limit:  limit,
			Status: st,
			Name:   name,
		},
	}, nil
}

func encodeFileDownloadResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	switch resp := response.(type) {
	case downloadReportResp:
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", resp.File.Name))
		w.Header().Set("Content-Type", resp.File.Format.ContentType())
		_, err := w.Write(resp.File.Data)
		return err
	default:
		if ar, ok := response.(supermq.Response); ok {
			for k, v := range ar.Headers() {
				w.Header().Set(k, v)
			}
			w.Header().Set("Content-Type", api.ContentType)
			w.WriteHeader(ar.Code())

			if ar.Empty() {
				return nil
			}
		}
		return json.NewEncoder(w).Encode(response)
	}
}
