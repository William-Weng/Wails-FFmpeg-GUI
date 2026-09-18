export const Containers = {
    MP4: "mp4",
    MKV: "mkv",
    TS: "ts",
    MP3: "mp3",
    AAC: "aac",
    OGG: "ogg",
    OPUS: "opus",
} as const;

export const VideoCodecs = {
    Copy: "copy",
    H264: "h264",
    H265: "h265",
} as const;

export type Container = typeof Containers[keyof typeof Containers];
export type VideoCodec = typeof VideoCodecs[keyof typeof VideoCodecs];