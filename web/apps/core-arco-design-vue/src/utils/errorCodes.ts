// 后端业务错误码枚举（F21：收敛魔法值）。
// 与 shared/errs 契约对齐：2001/2002 会话失效、2101 需要两步验证、5002 agent 不可用等。
export const ERR = {
  /** 登录失效（token 过期/版本变更） */
  Unauthorized: 2001,
  /** token 校验失败 */
  TokenInvalid: 2002,
  /** 请求参数错误 */
  BadRequest: 1001,
  /** 需要两步验证码 */
  OtpRequired: 2101,
  /** 节点未启用 Docker */
  AgentDisabled: 5002,
  /** agent 不可达 */
  AgentUnreach: 5001,
  /** 文件/操作执行失败 */
  FileOpFailed: 3004,
  /** 资源冲突 */
  Conflict: 3003,
} as const

export function isSessionError(code?: number) {
  return code === ERR.Unauthorized || code === ERR.TokenInvalid
}
