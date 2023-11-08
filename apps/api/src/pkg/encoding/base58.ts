import baseX from "base-x"

const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
const base58 = baseX(alphabet)


export const encode = base58.encode
export const decode = base58.decode
