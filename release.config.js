module.exports = {
    branches: [
        'main',
    ],
    preset: "conventionalcommits",
    plugins: [
        [
            '@semantic-release/commit-analyzer',
            {
                releaseRules: [
                    { "type": "patch", "release": "patch" },
                    { "type": "minor", "release": "minor" },
                    { "type": "major", "release": "major" },
                    { "scope": "no-release", "release": false },
                ],
            }
        ],
        [
            '@semantic-release/release-notes-generator',
            {
                presetConfig: {
                    types: [
                        { type: "patch", section: "Other" },
                        { type: "minor", section: "Other" },
                        { type: "major", section: "Other" },
                    ],
                },
            }
        ],
        [
            '@semantic-release/github',
            {
                assets: [
                    { path: 'ecs-events-exporter', label: 'ecs-events-exporter-linux-${nextRelease.gitTag}' }
                ],
                releaseBodyTemplate: "<%= nextRelease.notes %> \n\
### Image \n\
- [ghcr.io/leigholiver/ecs-events-exporter:<%= nextRelease.version %>](https://ghcr.io/leigholiver/ecs-events-exporter) \n\
- `docker pull ghcr.io/leigholiver/ecs-events-exporter:<%= nextRelease.version %>`",
            },
        ],
    ],
    verifyConditions: [
        '@semantic-release/github'
    ],
};
