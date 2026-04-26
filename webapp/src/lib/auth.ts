import NextAuth from "next-auth"
import GitHub from "next-auth/providers/github"
import { isAllowedUser } from "./auth-helpers"

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [GitHub],
  pages: {
    signIn: "/login",
  },
  callbacks: {
    signIn({ profile }) {
      return isAllowedUser(profile?.id)
    },
  },
})
