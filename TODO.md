# TODO

- Dashboard: daily/weekly expense chart with a by-category breakdown, using
  shadcn's Chart component (Recharts). Needs a new backend report endpoint
  for daily-granularity expense data, grouped by category — nothing finer
  than month/year exists yet (`/api/reports/month/:month`,
  `/api/reports/year/:year`).
- Dashboard: switch the tax/income/expense projections
  (`web/src/Dashboard.tsx`) from simple run-rate extrapolation (YTD ÷ months
  elapsed × 12) to a "same pace as last year" method — compare this year's
  YTD to last year's YTD-at-the-same-point, then apply last year's full-year
  growth ratio.
