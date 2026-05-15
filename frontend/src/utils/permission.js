export function isAdmin(user) {
  return user?.role === 'admin'
}

export function canWrite(user) {
  return user?.role === 'admin' || user?.role === 'developer'
}
