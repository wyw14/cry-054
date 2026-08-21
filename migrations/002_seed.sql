INSERT INTO grant_projects(id, name, year, annual_limit, version, created_at)
VALUES ('project-care-2026', '困难家庭医疗补助', 2026, 50000.00, 1, now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO claimants(id, display_name, identity_digest, plan_code, active, created_at)
VALUES ('claimant-demo', '演示申请人', 'sha256:demo-identity', 'enhanced', true, now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO rule_versions(id, project_id, version, effective_from, effective_to, cap, segments, conditions, published_at)
VALUES (
  'rule-care-2026-v1', 'project-care-2026', 1, DATE '2026-01-01', NULL, 30000.00,
  '[{"threshold":"10000.00","rate":"0.80"},{"threshold":"30000.00","rate":"0.50"}]'::jsonb,
  '{"categories":["medical","rehabilitation"],"plans":["basic","enhanced"]}'::jsonb,
  now()
)
ON CONFLICT (project_id, version) DO NOTHING;

