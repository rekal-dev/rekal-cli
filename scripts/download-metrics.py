"""Snapshot public GitHub asset counters; these are not unique installs."""
import datetime
import json
import os
from pathlib import Path
import urllib.request

assets = []
page = 1
while True:
    request = urllib.request.Request(
        f'https://api.github.com/repos/rekal-dev/rekal-cli/releases?per_page=100&page={page}',
        headers={'Accept': 'application/vnd.github+json', 'User-Agent': 'rekal-download-metrics'},
    )
    with urllib.request.urlopen(request, timeout=30) as response:
        releases = json.load(response)
    for release in releases:
        for asset in release['assets']:
            assets.append({'tag': release['tag_name'], 'asset_id': asset['id'],
                           'name': asset['name'], 'downloads': asset['download_count'],
                           'created_at': asset['created_at']})
    if len(releases) < 100:
        break
    page += 1
archives = sum(a['downloads'] for a in assets if a['name'].startswith('rekal_') and a['name'].endswith('.tar.gz'))
snapshot = {'archive_downloads': archives, 'captured_at': datetime.datetime.now(datetime.timezone.utc).isoformat(),
            'repository': 'rekal-dev/rekal-cli', 'assets': assets}
Path('download-metrics.json').write_text(json.dumps(snapshot, indent=2) + '\n')
Path('downloads.json').write_text(json.dumps({'schemaVersion': 1, 'label': 'downloads', 'message': f'{archives:,}', 'color': '22d3ee'}) + '\n')
summary = f'## Release downloads\n\n{archives} cumulative archive downloads across {page} release API page(s).\n\nIncludes repeat downloads, CI and manual tests. Not unique users or completed installs.\n\n| Release | Asset | Downloads |\n| --- | --- | ---: |\n'
summary += ''.join(f"| {a['tag']} | {a['name']} | {a['downloads']} |\n" for a in assets)
if os.environ.get('GITHUB_STEP_SUMMARY'):
    Path(os.environ['GITHUB_STEP_SUMMARY']).write_text(summary)
print(f'Snapshotted {len(assets)} assets; {archives} archive downloads')
