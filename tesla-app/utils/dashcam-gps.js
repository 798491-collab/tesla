import { get } from '@/utils/request.js'
import { insertTrackBatch, getTracks, findNearestTrack } from '@/utils/dashcam-db.js'

function haversineDistance(lat1, lon1, lat2, lon2) {
  const R = 6371000
  const toRad = (d) => (d * Math.PI) / 180
  const dLat = toRad(lat2 - lat1)
  const dLon = toRad(lon2 - lon1)
  const a =
    Math.sin(dLat / 2) ** 2 +
    Math.cos(toRad(lat1)) * Math.cos(toRad(lat2)) * Math.sin(dLon / 2) ** 2
  return R * 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
}

export function fetchTracksFromCloud(vin, startTime, endTime) {
  return get(`/api/vehicle/${vin}/tracks`, { start: startTime, end: endTime }).then(
    (res) => res.data || []
  )
}

export function cacheTracks(vin, tracks) {
  if (!tracks || !tracks.length) return Promise.resolve()
  const batch = tracks.map((t) => ({
    vin,
    latitude: t.latitude,
    longitude: t.longitude,
    speed: t.speed,
    timestamp: t.timestamp
  }))
  return insertTrackBatch(batch)
}

export function getCachedTracks(vin, startTime, endTime) {
  return getTracks({ vin, startTime, endTime }).then((cached) => {
    if (cached && cached.length) return cached
    return fetchTracksFromCloud(vin, startTime, endTime).then((tracks) => {
      if (tracks && tracks.length) {
        return cacheTracks(vin, tracks).then(() => tracks)
      }
      return []
    })
  })
}

export function fuseEventWithGPS(event) {
  return findNearestTrack(event.vin, event.event_time).then((track) => {
    if (track) {
      event.latitude = track.latitude
      event.longitude = track.longitude
    }
    return event
  })
}

export function batchFuseEvents(events) {
  if (!events || !events.length) return Promise.resolve([])
  return Promise.all(events.map((e) => fuseEventWithGPS(e)))
}

export function getTrackStats(vin, startTime, endTime) {
  return getCachedTracks(vin, startTime, endTime).then((tracks) => {
    if (!tracks || !tracks.length) {
      return { totalDistance: 0, avgSpeed: 0, maxSpeed: 0, duration: 0, pointCount: 0 }
    }
    let totalDistance = 0
    let maxSpeed = 0
    let speedSum = 0
    for (let i = 0; i < tracks.length; i++) {
      speedSum += tracks[i].speed || 0
      if ((tracks[i].speed || 0) > maxSpeed) maxSpeed = tracks[i].speed
      if (i > 0) {
        totalDistance += haversineDistance(
          tracks[i - 1].latitude,
          tracks[i - 1].longitude,
          tracks[i].latitude,
          tracks[i].longitude
        )
      }
    }
    const duration = tracks[tracks.length - 1].timestamp - tracks[0].timestamp
    return {
      totalDistance,
      avgSpeed: tracks.length ? speedSum / tracks.length : 0,
      maxSpeed,
      duration,
      pointCount: tracks.length
    }
  })
}
