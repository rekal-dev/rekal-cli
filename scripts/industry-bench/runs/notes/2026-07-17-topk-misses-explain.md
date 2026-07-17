# Top-5 Evidence misses (resolved within top-10) — explain dump

Generated with: `rekal --limit 10 --explain` on synthetic repos

Cases: 2

## Case 1: locomo conv-26 q5

- category: multi-hop
- gold: 'Transgender woman'
- evidence_session_ids: ['s1']
- evidence rank0 in top10: 5

**Question:**

> What is Caroline's identity?

**top5**

| rank0 | session_id | score | conf | mass | top files | bm25 | lsa | nomic | facet |
|---:|---|---:|---:|---:|---|---:|---:|---:|---:|
| 0 | 01KXQDFS4QPRS8ANCZNJA2MF8N | 0.95 | 0.57 | 2.13 | 2023-08-23-s13.md | 1 | 1 | 0.91 | 0 |
| 1 | 01KXQDFN6P4YBTPZK1NNEQ4BQJ | 0.87 | 0.62 | 1.49 | 2023-08-14-s11.md | 0.7 | 0.87 | 0.99 | 0 |
| 2 | 01KXQDEQYGQZW8F7TDQ54143T5 | 0.82 | 0.61 | 1.31 | 2023-06-09-s3.md | 0.61 | 0.69 | 0.97 | 0 |
| 3 | 01KXQDG79RJG1CJQ8BG2RKXYZA | 0.8 | 0.57 | 1.42 | 2023-09-13-s16.md | 0.67 | 0.66 | 0.91 | 0 |
| 4 | 01KXQDGA9WMRJRDECR1ENQG3B3 | 0.72 | 0.56 | 1.42 | 2023-10-13-s17.md | 0.67 | 0 | 0.89 | 0 |

**top10**

| rank0 | session_id | score | conf | mass | top files | bm25 | lsa | nomic | facet |
|---:|---|---:|---:|---:|---|---:|---:|---:|---:|
| 0 | 01KXQDFS4QPRS8ANCZNJA2MF8N | 0.95 | 0.57 | 2.13 | 2023-08-23-s13.md | 1 | 1 | 0.91 | 0 |
| 1 | 01KXQDFN6P4YBTPZK1NNEQ4BQJ | 0.87 | 0.62 | 1.49 | 2023-08-14-s11.md | 0.7 | 0.87 | 0.99 | 0 |
| 2 | 01KXQDEQYGQZW8F7TDQ54143T5 | 0.82 | 0.61 | 1.31 | 2023-06-09-s3.md | 0.61 | 0.69 | 0.97 | 0 |
| 3 | 01KXQDG79RJG1CJQ8BG2RKXYZA | 0.8 | 0.57 | 1.42 | 2023-09-13-s16.md | 0.67 | 0.66 | 0.91 | 0 |
| 4 | 01KXQDGA9WMRJRDECR1ENQG3B3 | 0.72 | 0.56 | 1.42 | 2023-10-13-s17.md | 0.67 | 0 | 0.89 | 0 |
| 5 | 01KXQDEMP944XH6PPA1R2PG9KG | 0.59 | 0.63 | 0.13 | 2023-05-08-s1.md | 0.06 | 0.15 | 1 | 0 |
| 6 | 01KXQDF6DS3Z3VKQVBNYRBM965 | 0.57 | 0.62 | 0.12 | 2023-07-06-s6.md | 0.06 | 0.06 | 1 | 0 |
| 7 | 01KXQDETZRX8HPYDXWBN4YBF8W | 0.57 | 0.63 | 0.12 | 2023-06-27-s4.md | 0.06 | 0 | 1 | 0 |
| 8 | 01KXQDGEREHFBVYHT7YRK2B8J9 | 0.56 | 0.61 | 0.14 | 2023-10-22-s19.md | 0.07 | 0 | 0.98 | 0 |
| 9 | 01KXQDF3E7DV1DTG93T5P53PRY | 0.55 | 0.61 | 0.11 | 2023-07-03-s5.md | 0.05 | 0.04 | 0.97 | 0 |

## Case 2: locomo conv-26 q8

- category: multi-hop
- gold: 'Single'
- evidence_session_ids: ['s2', 's3']
- evidence rank0 in top10: 8

**Question:**

> What is Caroline's relationship status?

**top5**

| rank0 | session_id | score | conf | mass | top files | bm25 | lsa | nomic | facet |
|---:|---|---:|---:|---:|---|---:|---:|---:|---:|
| 0 | 01KXQDG79RJG1CJQ8BG2RKXYZA | 0.85 | 0.57 | 2.18 | 2023-09-13-s16.md | 1 | 0.18 | 0.89 | 0 |
| 1 | 01KXQDEMP944XH6PPA1R2PG9KG | 0.67 | 0.71 | 0.13 | 2023-05-08-s1.md | 0.06 | 1 | 0.99 | 0 |
| 2 | 01KXQDF6DS3Z3VKQVBNYRBM965 | 0.62 | 0.64 | 0.12 | 2023-07-06-s6.md | 0.05 | 0.55 | 1 | 0 |
| 3 | 01KXQDFG0K6B79PBX9Z2QSR2GS | 0.6 | 0.61 | 0.12 | 2023-07-17-s9.md | 0.05 | 0.58 | 0.94 | 0 |
| 4 | 01KXQDF3E7DV1DTG93T5P53PRY | 0.59 | 0.61 | 0.11 | 2023-07-03-s5.md | 0.05 | 0.53 | 0.94 | 0 |

**top10**

| rank0 | session_id | score | conf | mass | top files | bm25 | lsa | nomic | facet |
|---:|---|---:|---:|---:|---|---:|---:|---:|---:|
| 0 | 01KXQDG79RJG1CJQ8BG2RKXYZA | 0.85 | 0.57 | 2.18 | 2023-09-13-s16.md | 1 | 0.18 | 0.89 | 0 |
| 1 | 01KXQDEMP944XH6PPA1R2PG9KG | 0.67 | 0.71 | 0.13 | 2023-05-08-s1.md | 0.06 | 1 | 0.99 | 0 |
| 2 | 01KXQDF6DS3Z3VKQVBNYRBM965 | 0.62 | 0.64 | 0.12 | 2023-07-06-s6.md | 0.05 | 0.55 | 1 | 0 |
| 3 | 01KXQDFG0K6B79PBX9Z2QSR2GS | 0.6 | 0.61 | 0.12 | 2023-07-17-s9.md | 0.05 | 0.58 | 0.94 | 0 |
| 4 | 01KXQDF3E7DV1DTG93T5P53PRY | 0.59 | 0.61 | 0.11 | 2023-07-03-s5.md | 0.05 | 0.53 | 0.94 | 0 |
| 5 | 01KXQDFPYMNYJ8R0AEDNEAXHFB | 0.58 | 0.6 | 0.13 | 2023-08-17-s12.md | 0.06 | 0.46 | 0.94 | 0 |
| 6 | 01KXQDETZRX8HPYDXWBN4YBF8W | 0.58 | 0.62 | 0.12 | 2023-06-27-s4.md | 0.06 | 0.35 | 0.96 | 0 |
| 7 | 01KXQDGEREHFBVYHT7YRK2B8J9 | 0.58 | 0.59 | 0.14 | 2023-10-22-s19.md | 0.07 | 0.52 | 0.91 | 0 |
| 8 | 01KXQDEPC6X1P7GSR6SCW0PFCF | 0.57 | 0.61 | 0.11 | 2023-05-25-s2.md | 0.05 | 0.39 | 0.94 | 0 |
| 9 | 01KXQDFH0QBZZDG24VABWN3AK4 | 0.56 | 0.6 | 0.14 | 2023-07-20-s10.md | 0.06 | 0.26 | 0.94 | 0 |

