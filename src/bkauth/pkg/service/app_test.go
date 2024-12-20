/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - Auth服务(BlueKing - Auth) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package service

import (
	"errors"

	"github.com/agiledragon/gomonkey"
	. "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"bkauth/pkg/database"
	"bkauth/pkg/database/dao"
	"bkauth/pkg/database/dao/mock"
	"bkauth/pkg/service/types"
)

var _ = Describe("App", func() {
	Describe("Exists cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().Exists("bkauth").Return(true, nil)

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}
			exists, err := svc.Exists("bkauth")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, exists)
		})

		It("error", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().Exists("bkauth").Return(false, errors.New("error"))

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			_, err := svc.Exists("bkauth")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("NameExists cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().NameExists("bkauth").Return(true, nil)

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}
			exists, err := svc.NameExists("bkauth")
			assert.NoError(GinkgoT(), err)
			assert.Equal(GinkgoT(), true, exists)
		})

		It("error", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().NameExists("bkauth").Return(false, errors.New("error"))

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			_, err := svc.NameExists("bkauth")
			assert.Error(GinkgoT(), err)
		})
	})

	Describe("Create cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().CreateWithTx(gomock.Any(), dao.App{
				Code: "bkauth", Name: "bkauth", Description: "bkauth intro",
			}).Return(nil)

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAccessKeyManager.EXPECT().CreateWithTx(gomock.Any(), dao.AccessKey{
				AppCode:       "bkauth",
				AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				CreatedSource: "bk_paas",
			}).Return(int64(1), nil)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()

			patches := gomonkey.ApplyFunc(database.GenerateDefaultDBTx, db.Beginx)
			defer patches.Reset()

			patches.ApplyFunc(newDaoAccessKey, func(_, _ string) dao.AccessKey {
				return dao.AccessKey{
					AppCode:       "bkauth",
					AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
					CreatedSource: "bk_paas",
				}
			})

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			err := svc.Create(types.App{Code: "bkauth", Name: "bkauth", Description: "bkauth intro"}, "bk_paas")
			assert.NoError(GinkgoT(), err)

			err = dbMock.ExpectationsWereMet()
			assert.NoError(GinkgoT(), err)
		})

		It("app create error", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().CreateWithTx(gomock.Any(), dao.App{
				Code: "bkauth", Name: "bkauth", Description: "bkauth intro",
			}).Return(errors.New("error"))

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()

			patches := gomonkey.ApplyFunc(database.GenerateDefaultDBTx, db.Beginx)
			defer patches.Reset()

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			err := svc.Create(types.App{Code: "bkauth", Name: "bkauth", Description: "bkauth intro"}, "bk_paas")
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "manager.CreateWithTx")
		})

		It("access key create error", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().CreateWithTx(gomock.Any(), dao.App{
				Code: "bkauth", Name: "bkauth", Description: "bkauth intro",
			}).Return(nil)

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAccessKeyManager.EXPECT().CreateWithTx(gomock.Any(), dao.AccessKey{
				AppCode:       "bkauth",
				AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				CreatedSource: "bk_paas",
			}).Return(int64(0), errors.New("error"))

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()

			patches := gomonkey.ApplyFunc(database.GenerateDefaultDBTx, db.Beginx)
			defer patches.Reset()

			patches.ApplyFunc(newDaoAccessKey, func(_, _ string) dao.AccessKey {
				return dao.AccessKey{
					AppCode:       "bkauth",
					AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
					CreatedSource: "bk_paas",
				}
			})

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			err := svc.Create(types.App{Code: "bkauth", Name: "bkauth", Description: "bkauth intro"}, "bk_paas")
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "accessKeyManager.CreateWithTx")
		})
	})

	Describe("CreateWithSecret cases", func() {
		var ctl *gomock.Controller

		BeforeEach(func() {
			ctl = gomock.NewController(GinkgoT())
		})

		AfterEach(func() {
			ctl.Finish()
		})

		It("ok", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().CreateWithTx(gomock.Any(), dao.App{
				Code: "bkauth", Name: "bkauth", Description: "bkauth intro",
			}).Return(nil)

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAccessKeyManager.EXPECT().CreateWithTx(gomock.Any(), dao.AccessKey{
				AppCode:       "bkauth",
				AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				CreatedSource: "bk_paas",
			}).Return(int64(1), nil)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()

			patches := gomonkey.ApplyFunc(database.GenerateDefaultDBTx, db.Beginx)
			defer patches.Reset()

			patches.ApplyFunc(newDaoAccessKeyWithAppSecret, func(_, _, _ string) dao.AccessKey {
				return dao.AccessKey{
					AppCode:       "bkauth",
					AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
					CreatedSource: "bk_paas",
				}
			})

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			err := svc.CreateWithSecret(
				types.App{Code: "bkauth", Name: "bkauth", Description: "bkauth intro"},
				"4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				"bk_paas",
			)
			assert.NoError(GinkgoT(), err)

			err = dbMock.ExpectationsWereMet()
			assert.NoError(GinkgoT(), err)
		})

		It("app create error", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().CreateWithTx(gomock.Any(), dao.App{
				Code: "bkauth", Name: "bkauth", Description: "bkauth intro",
			}).Return(errors.New("error"))

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()

			patches := gomonkey.ApplyFunc(database.GenerateDefaultDBTx, db.Beginx)
			defer patches.Reset()

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			err := svc.CreateWithSecret(
				types.App{Code: "bkauth", Name: "bkauth", Description: "bkauth intro"},
				"4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				"bk_paas",
			)
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "manager.CreateWithTx")
		})

		It("access key create error", func() {
			mockAppManager := mock.NewMockAppManager(ctl)
			mockAppManager.EXPECT().CreateWithTx(gomock.Any(), dao.App{
				Code: "bkauth", Name: "bkauth", Description: "bkauth intro",
			}).Return(nil)

			mockAccessKeyManager := mock.NewMockAccessKeyManager(ctl)
			mockAccessKeyManager.EXPECT().CreateWithTx(gomock.Any(), dao.AccessKey{
				AppCode:       "bkauth",
				AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				CreatedSource: "bk_paas",
			}).Return(int64(0), errors.New("error"))

			db, dbMock := database.NewMockSqlxDB()
			dbMock.ExpectBegin()
			dbMock.ExpectCommit()

			patches := gomonkey.ApplyFunc(database.GenerateDefaultDBTx, db.Beginx)
			defer patches.Reset()

			patches.ApplyFunc(newDaoAccessKeyWithAppSecret, func(_, _, _ string) dao.AccessKey {
				return dao.AccessKey{
					AppCode:       "bkauth",
					AppSecret:     "4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
					CreatedSource: "bk_paas",
				}
			})

			svc := appService{
				manager:          mockAppManager,
				accessKeyManager: mockAccessKeyManager,
			}

			err := svc.CreateWithSecret(
				types.App{Code: "bkauth", Name: "bkauth", Description: "bkauth intro"},
				"4d7a-b6b8-f3c255fff041-a59ddb37-94ae",
				"bk_paas",
			)
			assert.Error(GinkgoT(), err)
			assert.Contains(GinkgoT(), err.Error(), "accessKeyManager.CreateWithTx")
		})
	})
})
